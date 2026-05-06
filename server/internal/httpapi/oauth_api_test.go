package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

func setupTestOAuthAPI(t *testing.T) (*OAuthAPI, *auth.OAuthServerService, *database.UserStore, *database.OAuthStore, config.AuthConfig) {
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
		AllowedOrigins: []string{
			"https://chatgpt.com",
			"https://chat.openai.com",
		},
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

	oauthService := auth.NewOAuthServerService(mcpCfg, authCfg, userStore, oauthStore, auth.NewService(userStore, authCfg, logger), logger)
	if oauthService == nil {
		t.Fatal("expected OAuth service to be created")
	}
	return NewOAuthAPI(oauthService, logger), oauthService, userStore, oauthStore, authCfg
}

func TestOAuthAPI_OpenIDConfiguration(t *testing.T) {
	api, _, _, _, _ := setupTestOAuthAPI(t)

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

func TestOAuthAPI_OpenIDConfigurationIncludesCORSForConfiguredOrigin(t *testing.T) {
	api, _, _, _, _ := setupTestOAuthAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	request.Header.Set("Origin", "https://chatgpt.com")
	responseRecorder := httptest.NewRecorder()

	api.handleOpenIDConfiguration(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}
	assertCORSHeaders(t, responseRecorder.Result(), "https://chatgpt.com", "GET, OPTIONS", oauthCORSDefaultAllowedHeaders)
}

func TestOAuthAPI_OAuthEndpointPreflightAllowsConfiguredOrigin(t *testing.T) {
	api, _, _, _, _ := setupTestOAuthAPI(t)

	tests := []struct {
		name         string
		path         string
		method       string
		wantAllow    string
		handle       func(http.ResponseWriter, *http.Request)
		requestHeads string
	}{
		{name: "openid", path: "/.well-known/openid-configuration", method: http.MethodOptions, wantAllow: "GET, OPTIONS", handle: api.handleOpenIDConfiguration, requestHeads: "authorization, content-type"},
		{name: "jwks", path: "/oauth/jwks.json", method: http.MethodOptions, wantAllow: "GET, OPTIONS", handle: api.handleJWKS, requestHeads: "authorization, content-type"},
		{name: "register", path: "/oauth/register", method: http.MethodOptions, wantAllow: "POST, OPTIONS", handle: api.handleRegisterClient, requestHeads: "content-type, x-custom-header"},
		{name: "authorize", path: "/oauth/authorize", method: http.MethodOptions, wantAllow: "GET, POST, OPTIONS", handle: api.handleAuthorize, requestHeads: "content-type"},
		{name: "token", path: "/oauth/token", method: http.MethodOptions, wantAllow: "POST, OPTIONS", handle: api.handleToken, requestHeads: "content-type"},
		{name: "google-callback", path: "/oauth/google/callback", method: http.MethodOptions, wantAllow: "GET, OPTIONS", handle: api.handleGoogleCallback, requestHeads: "content-type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			request.Header.Set("Origin", "https://chatgpt.com")
			request.Header.Set("Access-Control-Request-Method", strings.Split(tt.wantAllow, ",")[0])
			request.Header.Set("Access-Control-Request-Headers", tt.requestHeads)
			responseRecorder := httptest.NewRecorder()

			tt.handle(responseRecorder, request)

			if responseRecorder.Code != http.StatusNoContent {
				t.Fatalf("expected HTTP 204, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
			}
			if got := responseRecorder.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("expected Allow header %q, got %q", tt.wantAllow, got)
			}
			assertCORSHeaders(t, responseRecorder.Result(), "https://chatgpt.com", tt.wantAllow, tt.requestHeads)
			if !headerListContains(responseRecorder.Result().Header.Values("Vary"), "Access-Control-Request-Headers") {
				t.Fatalf("expected Vary to contain %q, got %q", "Access-Control-Request-Headers", responseRecorder.Result().Header.Values("Vary"))
			}
		})
	}
}

func TestOAuthAPI_OAuthEndpointRejectsDisallowedOrigin(t *testing.T) {
	api, _, _, _, _ := setupTestOAuthAPI(t)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		handle func(http.ResponseWriter, *http.Request)
	}{
		{name: "openid", method: http.MethodGet, path: "/.well-known/openid-configuration", handle: api.handleOpenIDConfiguration},
		{name: "jwks", method: http.MethodGet, path: "/oauth/jwks.json", handle: api.handleJWKS},
		{name: "register", method: http.MethodPost, path: "/oauth/register", body: `{"client_name":"x"}`, handle: api.handleRegisterClient},
		{name: "authorize-get", method: http.MethodGet, path: "/oauth/authorize?response_type=code", handle: api.handleAuthorize},
		{name: "authorize-post", method: http.MethodPost, path: "/oauth/authorize", body: "decision=approve", handle: api.handleAuthorize},
		{name: "token", method: http.MethodPost, path: "/oauth/token", body: "grant_type=authorization_code", handle: api.handleToken},
		{name: "google-callback", method: http.MethodGet, path: "/oauth/google/callback?code=test&state=test", handle: api.handleGoogleCallback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			request.Header.Set("Origin", "https://evil.example")
			if tt.method == http.MethodPost {
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			responseRecorder := httptest.NewRecorder()

			tt.handle(responseRecorder, request)

			if responseRecorder.Code != http.StatusForbidden {
				t.Fatalf("expected HTTP 403, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
			}
			if got := responseRecorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("expected no Access-Control-Allow-Origin header, got %q", got)
			}
		})
	}
}

func TestOAuthAPI_AuthorizeRedirectsToGoogleWithoutSession(t *testing.T) {
	api, oauthService, _, _, _ := setupTestOAuthAPI(t)
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
			if !cookie.Secure {
				t.Fatal("expected pending OAuth cookie to be secure for HTTPS issuer")
			}
			if cookie.SameSite != http.SameSiteNoneMode {
				t.Fatalf("expected pending OAuth cookie SameSite=None, got %v", cookie.SameSite)
			}
		}
	}
	if !foundPendingCookie {
		t.Fatal("expected pending OAuth cookie to be set")
	}
}

func TestOAuthAPI_AuthorizeShowsConsentPageForSignedInUser(t *testing.T) {
	api, oauthService, userStore, _, authCfg := setupTestOAuthAPI(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	registration, err := oauthService.RegisterClient(ctx, auth.OAuthDynamicClientRegistrationRequest{
		ClientName:   "ChatGPT Test Connector",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	requestURL := "/oauth/authorize?response_type=code&client_id=" + url.QueryEscape(registration.ClientID) +
		"&redirect_uri=" + url.QueryEscape(registration.RedirectURIs[0]) +
		"&scope=flyingforge.read&state=opaque-state&code_challenge=testchallenge&code_challenge_method=S256&resource=" + url.QueryEscape("https://flyingforge.example/mcp")
	request := httptest.NewRequest(http.MethodGet, requestURL, nil)
	request.AddCookie(makeSessionCookie(t, authCfg.JWTSecret, user.ID, time.Now().Add(time.Hour)))

	responseRecorder := httptest.NewRecorder()
	api.handleAuthorize(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if contentType := responseRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected HTML consent page, got content type %q", contentType)
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Allow <span class=\"app-name\">ChatGPT Test Connector</span> to access FlyingForge?") {
		t.Fatalf("expected consent prompt to mention client name, got body %q", body)
	}
	if !strings.Contains(body, "ChatGPT Test Connector is requesting access to your FlyingForge account.") {
		t.Fatalf("expected consent prompt to use generic access copy, got body %q", body)
	}
	if strings.Contains(body, "Client ID:") || strings.Contains(body, "Redirect URI:") || strings.Contains(body, "Requested scopes:") {
		t.Fatalf("expected consent prompt to hide raw client metadata, got body %q", body)
	}
	if !strings.Contains(body, "View your aircraft, receiver summaries, tuning, radios, and backup metadata.") {
		t.Fatalf("expected consent prompt to show human-readable access description, got body %q", body)
	}
	if !strings.Contains(body, "name=\"consent_token\"") {
		t.Fatalf("expected consent prompt to include a signed consent token, got body %q", body)
	}
	if !strings.Contains(body, "name=\"decision\" value=\"approve\"") {
		t.Fatalf("expected consent form approve button, got body %q", body)
	}
}

func TestOAuthAPI_AuthorizeApprovalRedirectsBackToClient(t *testing.T) {
	api, oauthService, userStore, _, authCfg := setupTestOAuthAPI(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	registration, err := oauthService.RegisterClient(ctx, auth.OAuthDynamicClientRegistrationRequest{
		ClientName:   "ChatGPT Test Connector",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	authRequest, err := oauthService.ParseAuthorizationRequest(url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{registration.ClientID},
		"redirect_uri":          []string{registration.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{"testchallenge"},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("parse auth request: %v", err)
	}
	consentToken, err := oauthService.BuildAuthorizationConsentToken(user.ID, authRequest)
	if err != nil {
		t.Fatalf("build consent token: %v", err)
	}

	form := url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{registration.ClientID},
		"redirect_uri":          []string{registration.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{"testchallenge"},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
		"consent_token":         []string{consentToken},
		"decision":              []string{"approve"},
	}
	request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(makeSessionCookie(t, authCfg.JWTSecret, user.ID, time.Now().Add(time.Hour)))

	responseRecorder := httptest.NewRecorder()
	api.handleAuthorize(responseRecorder, request)

	if responseRecorder.Code != http.StatusSeeOther {
		t.Fatalf("expected HTTP 303, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	location := responseRecorder.Header().Get("Location")
	if !strings.HasPrefix(location, registration.RedirectURIs[0]) {
		t.Fatalf("expected redirect back to client redirect URI, got %q", location)
	}
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if parsedLocation.Query().Get("code") == "" {
		t.Fatalf("expected authorization code in redirect query, got %q", location)
	}
	if parsedLocation.Query().Get("state") != "opaque-state" {
		t.Fatalf("expected state to round-trip, got %q", parsedLocation.Query().Get("state"))
	}
}

func TestOAuthAPI_AuthorizeErrorsRedirectToRegisteredClient(t *testing.T) {
	api, oauthService, userStore, oauthStore, authCfg := setupTestOAuthAPI(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	client, err := oauthStore.CreateClient(
		ctx,
		"ff_mcp_error_redirect_test",
		"Error Redirect Connector",
		[]string{"https://chat.openai.com/a/oauth/callback"},
		[]string{models.OAuthGrantTypeRefreshToken},
		[]string{models.OAuthResponseTypeCode},
		models.OAuthTokenEndpointAuthMethodNone,
		"flyingforge.read",
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	authRequest, err := oauthService.ParseAuthorizationRequest(url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{client.ClientID},
		"redirect_uri":          []string{client.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{"testchallenge"},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("parse auth request: %v", err)
	}
	consentToken, err := oauthService.BuildAuthorizationConsentToken(user.ID, authRequest)
	if err != nil {
		t.Fatalf("build consent token: %v", err)
	}

	form := url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{client.ClientID},
		"redirect_uri":          []string{client.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{"testchallenge"},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
		"consent_token":         []string{consentToken},
		"decision":              []string{"approve"},
	}
	request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(makeSessionCookie(t, authCfg.JWTSecret, user.ID, time.Now().Add(time.Hour)))

	responseRecorder := httptest.NewRecorder()
	api.handleAuthorize(responseRecorder, request)

	if responseRecorder.Code != http.StatusSeeOther {
		t.Fatalf("expected HTTP 303, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	location := responseRecorder.Header().Get("Location")
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if parsedLocation.Query().Get("error") != "unauthorized_client" {
		t.Fatalf("expected unauthorized_client redirect, got %q", location)
	}
	if parsedLocation.Query().Get("state") != "opaque-state" {
		t.Fatalf("expected state to round-trip on error redirect, got %q", location)
	}
}

func TestOAuthAPI_AuthorizeApprovalRequiresConsentToken(t *testing.T) {
	api, oauthService, userStore, _, authCfg := setupTestOAuthAPI(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	registration, err := oauthService.RegisterClient(ctx, auth.OAuthDynamicClientRegistrationRequest{
		ClientName:   "ChatGPT Test Connector",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	form := url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{registration.ClientID},
		"redirect_uri":          []string{registration.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{"testchallenge"},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
		"decision":              []string{"approve"},
	}
	request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(makeSessionCookie(t, authCfg.JWTSecret, user.ID, time.Now().Add(time.Hour)))

	responseRecorder := httptest.NewRecorder()
	api.handleAuthorize(responseRecorder, request)

	if responseRecorder.Code != http.StatusSeeOther {
		t.Fatalf("expected HTTP 303, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	location := responseRecorder.Header().Get("Location")
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if parsedLocation.Query().Get("error") != "invalid_request" {
		t.Fatalf("expected invalid_request redirect, got %q", location)
	}
}

func TestOAuthAPI_TokenResponsesDisableCaching(t *testing.T) {
	api, oauthService, userStore, _, _ := setupTestOAuthAPI(t)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	registration, err := oauthService.RegisterClient(ctx, auth.OAuthDynamicClientRegistrationRequest{
		ClientName:   "ChatGPT Test Connector",
		RedirectURIs: []string{"https://chat.openai.com/a/oauth/callback"},
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	verifier := "http-token-cache-verifier-1234567890"
	authRequest, err := oauthService.ParseAuthorizationRequest(url.Values{
		"response_type":         []string{"code"},
		"client_id":             []string{registration.ClientID},
		"redirect_uri":          []string{registration.RedirectURIs[0]},
		"scope":                 []string{"flyingforge.read"},
		"state":                 []string{"opaque-state"},
		"code_challenge":        []string{codeChallengeForVerifier(verifier)},
		"code_challenge_method": []string{"S256"},
		"resource":              []string{"https://flyingforge.example/mcp"},
	})
	if err != nil {
		t.Fatalf("parse auth request: %v", err)
	}
	redirectURL, err := oauthService.Authorize(ctx, authRequest, user.ID)
	if err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	code := mustCodeFromRedirectURL(t, redirectURL)

	form := url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{registration.ClientID},
		"code":          []string{code},
		"redirect_uri":  []string{registration.RedirectURIs[0]},
		"code_verifier": []string{verifier},
		"resource":      []string{"https://flyingforge.example/mcp"},
	}
	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://chatgpt.com")
	responseRecorder := httptest.NewRecorder()

	api.handleToken(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	assertNoStoreHeaders(t, responseRecorder.Result())
	assertCORSHeaders(t, responseRecorder.Result(), "https://chatgpt.com", "POST, OPTIONS", oauthCORSDefaultAllowedHeaders)
}

func TestOAuthAPI_ErrorResponsesDisableCaching(t *testing.T) {
	api, _, _, _, _ := setupTestOAuthAPI(t)

	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader("%%%"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://chatgpt.com")
	responseRecorder := httptest.NewRecorder()

	api.handleToken(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d with body %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	assertNoStoreHeaders(t, responseRecorder.Result())
	assertCORSHeaders(t, responseRecorder.Result(), "https://chatgpt.com", "POST, OPTIONS", oauthCORSDefaultAllowedHeaders)
}

func TestDescribeAuthorizationAccess_SortsUnknownScopes(t *testing.T) {
	descriptions := describeAuthorizationAccess("scope.z flyingforge.read scope.a")
	want := []string{
		"View your aircraft, receiver summaries, tuning, radios, and backup metadata.",
		"Use read-only access only; this app cannot modify your FlyingForge data.",
		"Access scope: scope.a",
		"Access scope: scope.z",
	}

	if len(descriptions) != len(want) {
		t.Fatalf("expected %d descriptions, got %d: %#v", len(want), len(descriptions), descriptions)
	}
	for i, expected := range want {
		if descriptions[i] != expected {
			t.Fatalf("description %d: expected %q, got %q", i, expected, descriptions[i])
		}
	}
}

func makeSessionCookie(t *testing.T, jwtSecret, userID string, expiresAt time.Time) *http.Cookie {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"use": "mcp_oauth_session",
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": expiresAt.Unix(),
	})
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("sign session token: %v", err)
	}
	return &http.Cookie{Name: "ff_mcp_oauth_session", Value: signed, Path: "/oauth"}
}

func mustCodeFromRedirectURL(t *testing.T, redirectURL string) string {
	t.Helper()

	parsed, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("expected code in redirect URL %q", redirectURL)
	}
	return code
}

func assertNoStoreHeaders(t *testing.T, response *http.Response) {
	t.Helper()

	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", got)
	}
	if got := response.Header.Get("Pragma"); got != "no-cache" {
		t.Fatalf("expected Pragma no-cache, got %q", got)
	}
}

func assertCORSHeaders(t *testing.T, response *http.Response, origin, allowMethods, allowHeaders string) {
	t.Helper()

	if got := response.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", origin, got)
	}
	if got := response.Header.Get("Access-Control-Allow-Methods"); got != allowMethods {
		t.Fatalf("expected Access-Control-Allow-Methods %q, got %q", allowMethods, got)
	}
	if got := response.Header.Get("Access-Control-Allow-Headers"); got != allowHeaders {
		t.Fatalf("expected Access-Control-Allow-Headers %q, got %q", allowHeaders, got)
	}
	for _, varyValue := range []string{"Origin"} {
		if !headerListContains(response.Header.Values("Vary"), varyValue) {
			t.Fatalf("expected Vary to contain %q, got %q", varyValue, response.Header.Values("Vary"))
		}
	}
}

func headerListContains(values []string, expected string) bool {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), expected) {
				return true
			}
		}
	}
	return false
}

func codeChallengeForVerifier(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
