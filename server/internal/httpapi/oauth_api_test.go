package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

func setupTestOAuthAPI(t *testing.T) (*OAuthAPI, *auth.OAuthServerService) {
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

	oauthService := auth.NewOAuthServerService(mcpCfg, authCfg, userStore, oauthStore, auth.NewService(userStore, authCfg, logger), logger)
	if oauthService == nil {
		t.Fatal("expected OAuth service to be created")
	}
	return NewOAuthAPI(oauthService, logger), oauthService
}

func TestOAuthAPI_OpenIDConfiguration(t *testing.T) {
	api, _ := setupTestOAuthAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	responseRecorder := httptest.NewRecorder()

	api.handleOpenIDConfiguration(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}

	var metadata auth.OAuthAuthorizationServerMetadata
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("decode metadata response: %v", err)
	}
	if metadata.AuthorizationEndpoint != "https://flyingforge.example/oauth/authorize" {
		t.Fatalf("unexpected authorization endpoint: %+v", metadata)
	}
	if metadata.JWKSURI != "https://flyingforge.example/oauth/jwks.json" {
		t.Fatalf("unexpected JWKS URI: %+v", metadata)
	}
}

func TestOAuthAPI_AuthorizeRedirectsToGoogleWithoutSession(t *testing.T) {
	api, oauthService := setupTestOAuthAPI(t)
	ctx := context.Background()

	registration, err := oauthService.RegisterClient(ctx, auth.OAuthDynamicClientRegistrationRequest{
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	requestURL := "/oauth/authorize?response_type=code&client_id=" + url.QueryEscape(registration.ClientID) +
		"&redirect_uri=" + url.QueryEscape(registration.RedirectURIs[0]) +
		"&scope=flyingforge.read&state=opaque-state&code_challenge=testchallenge&code_challenge_method=S256&resource=" + url.QueryEscape("https://flyingforge.example/mcp")
	request := httptest.NewRequest(http.MethodGet, requestURL, nil)
	responseRecorder := httptest.NewRecorder()

	api.handleAuthorize(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("expected HTTP 302, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	location := responseRecorder.Header().Get("Location")
	if location == "" {
		t.Fatal("expected redirect location to Google OAuth")
	}
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if parsedLocation.Host != "accounts.google.com" {
		t.Fatalf("expected Google redirect host, got %q", parsedLocation.Host)
	}
	foundPendingCookie := false
	for _, cookie := range responseRecorder.Result().Cookies() {
		if cookie.Name == oauthService.PendingCookieName() {
			foundPendingCookie = true
			if cookie.Value == "" {
				t.Fatal("expected pending OAuth cookie to be populated")
			}
		}
	}
	if !foundPendingCookie {
		t.Fatal("expected pending OAuth cookie to be set")
	}
}
