package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

func setupTestOAuthServerService(t *testing.T) (*OAuthServerService, *database.UserStore) {
	t.Helper()

	testDB := testutil.NewTestDB(t)
	t.Cleanup(func() { testDB.Close() })
	t.Cleanup(func() { testDB.Cleanup(context.Background()) })

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)
	oauthStore := database.NewOAuthStore(db)
	logger := testutil.NullLogger()
	authCfg := config.AuthConfig{
		JWTSecret:          "test-secret-key-minimum-32-chars-long",
		JWTIssuer:          "flyingforge-test",
		JWTAudience:        "flyingforge-users",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    7 * 24 * time.Hour,
		GoogleClientID:     "google-client-id",
		GoogleClientSecret: "google-client-secret",
	}
	mcpCfg := config.MCPConfig{
		PublicBaseURL: "https://flyingforge.example",
		Auth: config.MCPAuthConfig{
			Enabled:              true,
			SelfHosted:           true,
			AllowEphemeralKey:    true,
			Issuer:               "https://flyingforge.example",
			Audience:             "https://flyingforge.example/mcp",
			Resource:             "https://flyingforge.example/mcp",
			RequiredScopes:       []string{"flyingforge.read"},
			GoogleRedirectURI:    "https://flyingforge.example/oauth/google/callback",
			AccessTokenTTL:       time.Hour,
			AuthorizationCodeTTL: 10 * time.Minute,
			RefreshTokenTTL:      24 * time.Hour,
			SessionTTL:           12 * time.Hour,
		},
	}

	googleAuth := NewService(userStore, authCfg, logger)
	service := NewOAuthServerService(mcpCfg, authCfg, userStore, oauthStore, googleAuth, logger)
	if service == nil {
		t.Fatal("expected self-hosted OAuth service to be created")
	}
	return service, userStore
}

func TestOAuthServerService_AuthorizationCodeAndRefreshFlow(t *testing.T) {
	service, userStore := setupTestOAuthServerService(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	registration, err := service.RegisterClient(ctx, OAuthDynamicClientRegistrationRequest{
		ClientName:   "ChatGPT Test Connector",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	verifier := "test-code-verifier-1234567890"
	challenge := codeChallengeForVerifier(verifier)
	requestValues := url.Values{}
	requestValues.Set("response_type", "code")
	requestValues.Set("client_id", registration.ClientID)
	requestValues.Set("redirect_uri", registration.RedirectURIs[0])
	requestValues.Set("scope", "flyingforge.read")
	requestValues.Set("state", "opaque-state")
	requestValues.Set("code_challenge", challenge)
	requestValues.Set("code_challenge_method", "S256")
	requestValues.Set("resource", "https://flyingforge.example/mcp")

	authRequest, err := service.ParseAuthorizationRequest(requestValues)
	if err != nil {
		t.Fatalf("parse auth request: %v", err)
	}

	redirectURL, err := service.Authorize(ctx, authRequest, user.ID)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	parsedRedirect, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	code := parsedRedirect.Query().Get("code")
	if code == "" {
		t.Fatalf("expected authorization code in redirect URL, got %q", redirectURL)
	}
	if parsedRedirect.Query().Get("state") != "opaque-state" {
		t.Fatalf("expected state round-trip in redirect URL, got %q", parsedRedirect.Query().Get("state"))
	}

	tokenResponse, err := service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{registration.ClientID},
		"code":          []string{code},
		"redirect_uri":  []string{registration.RedirectURIs[0]},
		"code_verifier": []string{verifier},
		"resource":      []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("exchange authorization code: %v", err)
	}
	if tokenResponse.AccessToken == "" || tokenResponse.RefreshToken == "" {
		t.Fatalf("expected access and refresh tokens, got %+v", tokenResponse)
	}
	if tokenResponse.Scope != "flyingforge.read" {
		t.Fatalf("expected scope flyingforge.read, got %q", tokenResponse.Scope)
	}
	if tokenResponse.Resource != "https://flyingforge.example/mcp" {
		t.Fatalf("expected resource to round-trip, got %q", tokenResponse.Resource)
	}
	if tokenResponse.ExpiresIn != int(time.Hour.Seconds()) {
		t.Fatalf("expected 1h expiry, got %d", tokenResponse.ExpiresIn)
	}

	refreshResponse, err := service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"refresh_token"},
		"client_id":     []string{registration.ClientID},
		"refresh_token": []string{tokenResponse.RefreshToken},
	})
	if err != nil {
		t.Fatalf("exchange refresh token: %v", err)
	}
	if refreshResponse.AccessToken == "" || refreshResponse.RefreshToken == "" {
		t.Fatalf("expected refreshed tokens, got %+v", refreshResponse)
	}
	if refreshResponse.RefreshToken == tokenResponse.RefreshToken {
		t.Fatalf("expected refresh token rotation")
	}
}

func TestOAuthServerService_GooglePendingStateAndSessionTokens(t *testing.T) {
	service, userStore := setupTestOAuthServerService(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	service.googleCodeResolver = func(context.Context, string, string) (*models.User, error) {
		return user, nil
	}

	googleURL, pendingToken, err := service.BuildGoogleAuthorizationURL("/oauth/authorize?client_id=abc")
	if err != nil {
		t.Fatalf("build Google authorization URL: %v", err)
	}
	parsedGoogleURL, err := url.Parse(googleURL)
	if err != nil {
		t.Fatalf("parse Google URL: %v", err)
	}
	state := parsedGoogleURL.Query().Get("state")
	if state == "" {
		t.Fatalf("expected Google state query parameter in %q", googleURL)
	}

	sessionToken, redirectTo, err := service.HandleGoogleCallback(ctx, "google-code", state, pendingToken)
	if err != nil {
		t.Fatalf("handle Google callback: %v", err)
	}
	if redirectTo != "/oauth/authorize?client_id=abc" {
		t.Fatalf("expected original authorize return path, got %q", redirectTo)
	}
	validatedUserID, err := service.ValidateSessionToken(sessionToken)
	if err != nil {
		t.Fatalf("validate session token: %v", err)
	}
	if validatedUserID != user.ID {
		t.Fatalf("expected session token subject %q, got %q", user.ID, validatedUserID)
	}
}

func TestOAuthServerService_RegisterClientRejectsNonHTTPSRedirectURIs(t *testing.T) {
	service, _ := setupTestOAuthServerService(t)

	_, err := service.RegisterClient(context.Background(), OAuthDynamicClientRegistrationRequest{
		RedirectURIs: []string{"http://localhost:3000/callback"},
	})
	if err == nil {
		t.Fatal("expected invalid_client_metadata error for non-HTTPS redirect URI")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected OAuthError, got %T", err)
	}
	if oauthErr.Code != "invalid_client_metadata" {
		t.Fatalf("expected invalid_client_metadata, got %q", oauthErr.Code)
	}
}

func TestOAuthServerService_InvalidAuthorizationCodeExchangeDoesNotBurnCode(t *testing.T) {
	service, userStore := setupTestOAuthServerService(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	clientA, err := service.RegisterClient(ctx, OAuthDynamicClientRegistrationRequest{
		ClientName:   "Connector A",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client A: %v", err)
	}
	clientB, err := service.RegisterClient(ctx, OAuthDynamicClientRegistrationRequest{
		ClientName:   "Connector B",
		RedirectURIs: []string{"https://chatgpt.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client B: %v", err)
	}

	verifier := "test-code-verifier-1234567890"
	challenge := codeChallengeForVerifier(verifier)
	authRequest, err := service.ParseAuthorizationRequest(url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{clientA.ClientID},
		"redirect_uri":          []string{clientA.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{challenge},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("parse auth request: %v", err)
	}

	redirectURL, err := service.Authorize(ctx, authRequest, user.ID)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	code := mustCodeFromRedirect(t, redirectURL)

	_, err = service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{clientB.ClientID},
		"code":          []string{code},
		"redirect_uri":  []string{clientB.RedirectURIs[0]},
		"code_verifier": []string{verifier},
		"resource":      []string{"https://flyingforge.example/mcp"},
	})
	if err == nil {
		t.Fatal("expected wrong-client authorization code exchange to fail")
	}

	tokenResponse, err := service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{clientA.ClientID},
		"code":          []string{code},
		"redirect_uri":  []string{clientA.RedirectURIs[0]},
		"code_verifier": []string{verifier},
		"resource":      []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("expected valid exchange after rejected attacker request: %v", err)
	}
	if tokenResponse.AccessToken == "" || tokenResponse.RefreshToken == "" {
		t.Fatalf("expected tokens from valid exchange, got %+v", tokenResponse)
	}
}

func TestOAuthServerService_InvalidRefreshRequestDoesNotRevokeToken(t *testing.T) {
	service, userStore := setupTestOAuthServerService(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	clientA, err := service.RegisterClient(ctx, OAuthDynamicClientRegistrationRequest{
		ClientName:   "Connector A",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client A: %v", err)
	}
	clientB, err := service.RegisterClient(ctx, OAuthDynamicClientRegistrationRequest{
		ClientName:   "Connector B",
		RedirectURIs: []string{"https://chatgpt.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client B: %v", err)
	}

	verifier := "refresh-verifier-1234567890"
	authRequest, err := service.ParseAuthorizationRequest(url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{clientA.ClientID},
		"redirect_uri":          []string{clientA.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{codeChallengeForVerifier(verifier)},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("parse auth request: %v", err)
	}

	redirectURL, err := service.Authorize(ctx, authRequest, user.ID)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	code := mustCodeFromRedirect(t, redirectURL)

	tokenResponse, err := service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{clientA.ClientID},
		"code":          []string{code},
		"redirect_uri":  []string{clientA.RedirectURIs[0]},
		"code_verifier": []string{verifier},
	})
	if err != nil {
		t.Fatalf("exchange authorization code: %v", err)
	}

	_, err = service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"refresh_token"},
		"client_id":     []string{clientB.ClientID},
		"refresh_token": []string{tokenResponse.RefreshToken},
	})
	if err == nil {
		t.Fatal("expected wrong-client refresh exchange to fail")
	}

	refreshResponse, err := service.ExchangeToken(ctx, url.Values{
		"grant_type":    []string{"refresh_token"},
		"client_id":     []string{clientA.ClientID},
		"refresh_token": []string{tokenResponse.RefreshToken},
	})
	if err != nil {
		t.Fatalf("expected valid refresh exchange after rejected attacker request: %v", err)
	}
	if refreshResponse.AccessToken == "" || refreshResponse.RefreshToken == "" {
		t.Fatalf("expected rotated tokens, got %+v", refreshResponse)
	}
}

func TestNewOAuthServerService_RequiresExplicitEphemeralKeyOptIn(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	t.Cleanup(func() { testDB.Close() })
	t.Cleanup(func() { testDB.Cleanup(context.Background()) })

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)
	oauthStore := database.NewOAuthStore(db)
	logger := testutil.NullLogger()
	authCfg := config.AuthConfig{
		JWTSecret:          "test-secret-key-minimum-32-chars-long",
		GoogleClientID:     "google-client-id",
		GoogleClientSecret: "google-client-secret",
	}
	mcpCfg := config.MCPConfig{
		PublicBaseURL: "https://flyingforge.example",
		Auth: config.MCPAuthConfig{
			Enabled:              true,
			SelfHosted:           true,
			Issuer:               "https://flyingforge.example",
			Audience:             "https://flyingforge.example/mcp",
			Resource:             "https://flyingforge.example/mcp",
			RequiredScopes:       []string{"flyingforge.read"},
			GoogleRedirectURI:    "https://flyingforge.example/oauth/google/callback",
			AccessTokenTTL:       time.Hour,
			AuthorizationCodeTTL: 10 * time.Minute,
			RefreshTokenTTL:      24 * time.Hour,
			SessionTTL:           12 * time.Hour,
		},
	}

	googleAuth := NewService(userStore, authCfg, logger)
	if service := NewOAuthServerService(mcpCfg, authCfg, userStore, oauthStore, googleAuth, logger); service != nil {
		t.Fatal("expected self-hosted OAuth service creation to fail without a signing key or explicit ephemeral-key opt-in")
	}

	mcpCfg.Auth.AllowEphemeralKey = true
	if service := NewOAuthServerService(mcpCfg, authCfg, userStore, oauthStore, googleAuth, logger); service == nil {
		t.Fatal("expected explicit ephemeral signing key opt-in to allow service creation")
	}
}

func mustCodeFromRedirect(t *testing.T, redirectURL string) string {
	t.Helper()

	parsedRedirect, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	code := parsedRedirect.Query().Get("code")
	if strings.TrimSpace(code) == "" {
		t.Fatalf("expected authorization code in redirect URL, got %q", redirectURL)
	}
	return code
}

func codeChallengeForVerifier(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
