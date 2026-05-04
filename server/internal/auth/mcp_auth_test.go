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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

type testOIDCProvider struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
	kid        string
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   provider.server.URL,
			"jwks_uri": provider.server.URL + "/keys",
		})
	})
	handler.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		pubKey := privateKey.PublicKey
		n := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(bigEndianBytes(pubKey.E))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": provider.kid,
					"use": "sig",
					"alg": "RS256",
					"n":   n,
					"e":   e,
				},
			},
		})
	})

	provider.server = httptest.NewServer(handler)
	t.Cleanup(provider.server.Close)

	return provider
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
