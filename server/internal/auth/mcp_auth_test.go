package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

type testOIDCProvider struct {
	server            *httptest.Server
	privateKey        *rsa.PrivateKey
	kid               string
	discoveryRequests atomic.Int32
	keysRequests      atomic.Int32
	discoveryHook     func(http.ResponseWriter, *http.Request)
	keysHook          func(http.ResponseWriter, *http.Request)
}

func newTestOIDCProvider(t *testing.T) *testOIDCProvider {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	provider := &testOIDCProvider{
		privateKey: privateKey,
		kid:        "test-kid",
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		provider.discoveryRequests.Add(1)
		if provider.discoveryHook != nil {
			provider.discoveryHook(w, r)
			return
		}
		provider.writeDiscoveryResponse(w)
	})
	handler.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		provider.keysRequests.Add(1)
		if provider.keysHook != nil {
			provider.keysHook(w, r)
			return
		}
		provider.writeKeysResponse(w)
	})

	provider.server = httptest.NewServer(handler)
	t.Cleanup(provider.server.Close)

	return provider
}

func (p *testOIDCProvider) writeDiscoveryResponse(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":   p.server.URL,
		"jwks_uri": p.server.URL + "/keys",
	})
}

func (p *testOIDCProvider) writeKeysResponse(w http.ResponseWriter) {
	pubKey := p.privateKey.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(bigEndianBytes(pubKey.E))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": p.kid,
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	})
}

func (p *testOIDCProvider) signToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = p.kid

	signed, err := token.SignedString(p.privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func bigEndianBytes(value int) []byte {
	if value == 0 {
		return []byte{0}
	}

	var bytes []byte
	for value > 0 {
		bytes = append([]byte{byte(value & 0xff)}, bytes...)
		value >>= 8
	}
	return bytes
}

func newTestMCPAuthService(provider *testOIDCProvider) *MCPAuthService {
	service := &MCPAuthService{
		cfg: config.MCPConfig{
			PublicBaseURL: "https://flyingforge.example.com",
			Auth: config.MCPAuthConfig{
				Enabled:        true,
				Issuer:         provider.server.URL,
				Audience:       "flyingforge-chatgpt",
				Resource:       "https://flyingforge.example.com/mcp",
				RequiredScopes: []string{"flyingforge.read"},
			},
		},
		logger: testutil.NullLogger(),
		client: provider.server.Client(),
	}

	return service
}

func baseClaims(issuer string) jwt.MapClaims {
	return jwt.MapClaims{
		"iss":            issuer,
		"sub":            "subject-123",
		"aud":            "flyingforge-chatgpt",
		"scope":          "flyingforge.read",
		"email":          "pilot@example.com",
		"email_verified": true,
		"exp":            time.Now().Add(1 * time.Hour).Unix(),
		"iat":            time.Now().Add(-1 * time.Minute).Unix(),
	}
}

func TestMCPAuthVerifyTokenAudienceAndScopeValidation(t *testing.T) {
	provider := newTestOIDCProvider(t)
	service := newTestMCPAuthService(provider)

	ctx := context.Background()

	validToken := provider.signToken(t, baseClaims(provider.server.URL))
	if _, err := service.verifyToken(ctx, validToken); err != nil {
		t.Fatalf("expected valid token to verify, got %v", err)
	}

	invalidAudienceClaims := baseClaims(provider.server.URL)
	invalidAudienceClaims["aud"] = "wrong-audience"
	invalidAudienceToken := provider.signToken(t, invalidAudienceClaims)

	_, err := service.verifyToken(ctx, invalidAudienceToken)
	if err == nil {
		t.Fatal("expected invalid audience token to fail verification")
	}
	authErr, ok := err.(*MCPAuthError)
	if !ok {
		t.Fatalf("expected MCPAuthError, got %T", err)
	}
	if authErr.Code != "invalid_token" {
		t.Fatalf("expected invalid_token code, got %+v", authErr)
	}

	missingScopeClaims := baseClaims(provider.server.URL)
	missingScopeClaims["scope"] = "profile"
	missingScopeToken := provider.signToken(t, missingScopeClaims)

	_, err = service.verifyToken(ctx, missingScopeToken)
	if err == nil {
		t.Fatal("expected missing-scope token to fail verification")
	}
	authErr, ok = err.(*MCPAuthError)
	if !ok {
		t.Fatalf("expected MCPAuthError, got %T", err)
	}
	if authErr.Code != "insufficient_scope" {
		t.Fatalf("expected insufficient_scope code, got %+v", authErr)
	}
}

func TestMCPAuthVerifyTokenAcceptsScopeAndScpClaimFormats(t *testing.T) {
	provider := newTestOIDCProvider(t)
	service := newTestMCPAuthService(provider)

	ctx := context.Background()

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "scope string",
			claims: func() jwt.MapClaims {
				claims := baseClaims(provider.server.URL)
				claims["scope"] = "flyingforge.read profile"
				return claims
			}(),
		},
		{
			name: "scope array",
			claims: func() jwt.MapClaims {
				claims := baseClaims(provider.server.URL)
				claims["scope"] = []string{"flyingforge.read", "profile"}
				return claims
			}(),
		},
		{
			name: "scp string",
			claims: func() jwt.MapClaims {
				claims := baseClaims(provider.server.URL)
				delete(claims, "scope")
				claims["scp"] = "flyingforge.read profile"
				return claims
			}(),
		},
		{
			name: "scp array",
			claims: func() jwt.MapClaims {
				claims := baseClaims(provider.server.URL)
				delete(claims, "scope")
				claims["scp"] = []string{"flyingforge.read", "profile"}
				return claims
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := provider.signToken(t, tt.claims)

			if _, err := service.verifyToken(ctx, token); err != nil {
				t.Fatalf("expected token with %s to verify, got %v", tt.name, err)
			}
		})
	}
}

func TestMCPAuthAuthenticateBearerTokenLinksVerifiedEmail(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	defer testDB.Close()

	ctx := context.Background()
	testDB.Cleanup(ctx)

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)
	logger := testutil.NullLogger()

	existingUser, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "Pilot",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}

	provider := newTestOIDCProvider(t)
	service := NewMCPAuthService(config.MCPConfig{
		PublicBaseURL: "https://flyingforge.example.com",
		Auth: config.MCPAuthConfig{
			Enabled:        true,
			Issuer:         provider.server.URL,
			Audience:       "flyingforge-chatgpt",
			Resource:       "https://flyingforge.example.com/mcp",
			RequiredScopes: []string{"flyingforge.read"},
		},
	}, userStore, logger)
	if service == nil {
		t.Fatal("expected MCP auth service to be initialized")
	}
	service.client = provider.server.Client()

	token := provider.signToken(t, baseClaims(provider.server.URL))

	userID, err := service.AuthenticateBearerToken(ctx, token)
	if err != nil {
		t.Fatalf("expected bearer token to authenticate, got %v", err)
	}
	if userID != existingUser.ID {
		t.Fatalf("expected existing user ID %s, got %s", existingUser.ID, userID)
	}

	identity, err := userStore.GetIdentityByProvider(ctx, models.AuthProviderMCPOAuth, provider.server.URL+"|subject-123")
	if err != nil {
		t.Fatalf("failed to load linked identity: %v", err)
	}
	if identity == nil {
		t.Fatal("expected linked MCP OAuth identity to be created")
	}
	if identity.UserID != existingUser.ID {
		t.Fatalf("expected identity to link to user %s, got %+v", existingUser.ID, identity)
	}

	user, err := userStore.GetByID(ctx, existingUser.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if user.LastLoginAt == nil {
		t.Fatal("expected AuthenticateBearerToken to update last_login_at")
	}
}

func TestMCPAuthAuthenticateBearerTokenRejectsIdentityLinkedToDifferentUser(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	defer testDB.Close()

	ctx := context.Background()
	testDB.Cleanup(ctx)

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)
	logger := testutil.NullLogger()

	emailMatchedUser, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "Pilot",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create email-matched user: %v", err)
	}

	linkedUser, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "other@example.com",
		DisplayName: "Other Pilot",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create linked user: %v", err)
	}

	provider := newTestOIDCProvider(t)
	subjectKey := provider.server.URL + "|subject-123"
	if _, err := userStore.CreateIdentity(ctx, linkedUser.ID, models.AuthProviderMCPOAuth, subjectKey, linkedUser.Email); err != nil {
		t.Fatalf("failed to seed linked identity: %v", err)
	}

	service := NewMCPAuthService(config.MCPConfig{
		PublicBaseURL: "https://flyingforge.example.com",
		Auth: config.MCPAuthConfig{
			Enabled:        true,
			Issuer:         provider.server.URL,
			Audience:       "flyingforge-chatgpt",
			Resource:       "https://flyingforge.example.com/mcp",
			RequiredScopes: []string{"flyingforge.read"},
		},
	}, userStore, logger)
	if service == nil {
		t.Fatal("expected MCP auth service to be initialized")
	}
	service.client = provider.server.Client()

	token := provider.signToken(t, baseClaims(provider.server.URL))

	_, err = service.AuthenticateBearerToken(ctx, token)
	if err == nil {
		t.Fatal("expected bearer token authentication to fail when the MCP identity is linked to another user")
	}

	authErr, ok := err.(*MCPAuthError)
	if !ok {
		t.Fatalf("expected MCPAuthError, got %T", err)
	}
	if authErr.Code != "invalid_token" {
		t.Fatalf("expected invalid_token code, got %+v", authErr)
	}
	if !strings.Contains(authErr.Message, "already linked to another FlyingForge account") {
		t.Fatalf("expected linked-account auth error, got %q", authErr.Message)
	}
	if emailMatchedUser.ID == linkedUser.ID {
		t.Fatal("expected distinct test users")
	}
}

func TestMCPAuthResolveUserAllowsDuplicateIdentityForSameUser(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	defer testDB.Close()

	ctx := context.Background()
	testDB.Cleanup(ctx)

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "Pilot",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}

	service := NewMCPAuthService(config.MCPConfig{
		PublicBaseURL: "https://flyingforge.example.com",
		Auth: config.MCPAuthConfig{
			Enabled:        true,
			RequiredScopes: []string{"flyingforge.read"},
		},
	}, userStore, testutil.NullLogger())
	if service == nil {
		t.Fatal("expected MCP auth service to be initialized")
	}

	lookupCalls := 0
	service.getIdentityByProvider = func(ctx context.Context, provider models.AuthProvider, subject string) (*models.UserIdentity, error) {
		lookupCalls++
		if lookupCalls == 1 {
			return nil, nil
		}
		return &models.UserIdentity{
			UserID:          user.ID,
			Provider:        provider,
			ProviderSubject: subject,
			ProviderEmail:   user.Email,
		}, nil
	}
	service.createIdentity = func(ctx context.Context, userID string, provider models.AuthProvider, subject, email string) (*models.UserIdentity, error) {
		if userID != user.ID {
			t.Fatalf("expected duplicate link check for user %s, got %s", user.ID, userID)
		}
		return nil, errors.New("identity already linked to another account")
	}

	userID, err := service.resolveUser(ctx, &mcpIdentity{
		Issuer:        "https://issuer.example.com",
		Subject:       "subject-123",
		Email:         "pilot@example.com",
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("expected duplicate identity for same user to succeed, got %v", err)
	}
	if userID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, userID)
	}
	if lookupCalls != 2 {
		t.Fatalf("expected 2 identity lookups, got %d", lookupCalls)
	}
}

func TestMCPAuthResolveUserRejectsDuplicateIdentityForDifferentUser(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	defer testDB.Close()

	ctx := context.Background()
	testDB.Cleanup(ctx)

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "Pilot",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}
	otherUser, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "other@example.com",
		DisplayName: "Other",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}

	service := NewMCPAuthService(config.MCPConfig{
		PublicBaseURL: "https://flyingforge.example.com",
		Auth: config.MCPAuthConfig{
			Enabled:        true,
			RequiredScopes: []string{"flyingforge.read"},
		},
	}, userStore, testutil.NullLogger())
	if service == nil {
		t.Fatal("expected MCP auth service to be initialized")
	}

	lookupCalls := 0
	service.getIdentityByProvider = func(ctx context.Context, provider models.AuthProvider, subject string) (*models.UserIdentity, error) {
		lookupCalls++
		if lookupCalls == 1 {
			return nil, nil
		}
		return &models.UserIdentity{
			UserID:          otherUser.ID,
			Provider:        provider,
			ProviderSubject: subject,
			ProviderEmail:   otherUser.Email,
		}, nil
	}
	service.createIdentity = func(ctx context.Context, userID string, provider models.AuthProvider, subject, email string) (*models.UserIdentity, error) {
		if userID != user.ID {
			t.Fatalf("expected duplicate link check for user %s, got %s", user.ID, userID)
		}
		return nil, errors.New("identity already linked to another account")
	}

	_, err = service.resolveUser(ctx, &mcpIdentity{
		Issuer:        "https://issuer.example.com",
		Subject:       "subject-123",
		Email:         "pilot@example.com",
		EmailVerified: true,
	})
	if err == nil {
		t.Fatal("expected duplicate identity for a different user to fail")
	}
	authErr, ok := err.(*MCPAuthError)
	if !ok {
		t.Fatalf("expected MCPAuthError, got %T", err)
	}
	if authErr.Code != "invalid_token" {
		t.Fatalf("expected invalid_token code, got %+v", authErr)
	}
	if !strings.Contains(authErr.Message, "already linked to another FlyingForge account") {
		t.Fatalf("expected linked-account auth error, got %q", authErr.Message)
	}
	if lookupCalls != 2 {
		t.Fatalf("expected 2 identity lookups, got %d", lookupCalls)
	}
}

func TestMCPAuthVerifyTokenSharesInFlightDiscoveryAndJWKSFetch(t *testing.T) {
	provider := newTestOIDCProvider(t)
	service := newTestMCPAuthService(provider)

	provider.keysHook = func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		provider.writeKeysResponse(w)
	}

	token := provider.signToken(t, baseClaims(provider.server.URL))

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.verifyToken(context.Background(), token)
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("expected concurrent token verifications to succeed, got %v", err)
		}
	}

	if got := provider.discoveryRequests.Load(); got != 1 {
		t.Fatalf("expected exactly 1 discovery request, got %d", got)
	}
	if got := provider.keysRequests.Load(); got != 1 {
		t.Fatalf("expected exactly 1 JWKS request, got %d", got)
	}
}

func TestMCPAuthGetSigningKeysWaiterHonorsContextCancellation(t *testing.T) {
	provider := newTestOIDCProvider(t)
	service := newTestMCPAuthService(provider)

	keysStarted := make(chan struct{})
	releaseKeys := make(chan struct{})
	provider.keysHook = func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-keysStarted:
		default:
			close(keysStarted)
		}
		<-releaseKeys
		provider.writeKeysResponse(w)
	}

	firstErr := make(chan error, 1)
	go func() {
		_, err := service.getSigningKeys(context.Background())
		firstErr <- err
	}()

	<-keysStarted

	var releaseOnce sync.Once
	time.AfterFunc(500*time.Millisecond, func() {
		releaseOnce.Do(func() {
			close(releaseKeys)
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := service.getSigningKeys(ctx)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded while waiting for signing keys, got %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("expected waiting caller to return promptly after context cancellation, took %s", elapsed)
	}

	releaseOnce.Do(func() {
		close(releaseKeys)
	})

	if err := <-firstErr; err != nil {
		t.Fatalf("expected initial signing key fetch to succeed, got %v", err)
	}
	if got := provider.discoveryRequests.Load(); got != 1 {
		t.Fatalf("expected exactly 1 discovery request, got %d", got)
	}
	if got := provider.keysRequests.Load(); got != 1 {
		t.Fatalf("expected exactly 1 JWKS request, got %d", got)
	}
}

func TestMCPAuthChallengeFormatsBearerHeader(t *testing.T) {
	service := &MCPAuthService{
		cfg: config.MCPConfig{
			PublicBaseURL: "https://flyingforge.example.com",
			Auth: config.MCPAuthConfig{
				Enabled:        true,
				Issuer:         "https://issuer.example.com",
				RequiredScopes: []string{"flyingforge.read"},
			},
		},
	}

	challenge := service.Challenge("invalid_token", `needs "quotes"`)
	if !strings.HasPrefix(challenge, "Bearer ") {
		t.Fatalf("expected Bearer challenge, got %q", challenge)
	}
	if strings.Contains(challenge, `Bearer,`) {
		t.Fatalf("expected no comma immediately after Bearer, got %q", challenge)
	}
	if !strings.Contains(challenge, `error="invalid_token"`) {
		t.Fatalf("expected error code in challenge, got %q", challenge)
	}
	if !strings.Contains(challenge, `error_description="needs 'quotes'"`) {
		t.Fatalf("expected sanitized description in challenge, got %q", challenge)
	}
}
