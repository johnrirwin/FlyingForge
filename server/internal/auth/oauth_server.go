package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	oauthGoogleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	oauthSessionCookieName  = "ff_mcp_oauth_session"
	oauthPendingCookieName  = "ff_mcp_oauth_pending"
	oauthStateTTL           = 15 * time.Minute
)

type OAuthError struct {
	Code        string
	Description string
	StatusCode  int
}

func (e *OAuthError) Error() string {
	if strings.TrimSpace(e.Description) != "" {
		return e.Description
	}
	return e.Code
}

func NormalizeOAuthError(err error) *OAuthError {
	if err == nil {
		return &OAuthError{Code: "server_error", Description: "unknown OAuth error", StatusCode: 500}
	}
	typed, ok := err.(*OAuthError)
	if ok {
		if typed.StatusCode == 0 {
			typed.StatusCode = 400
		}
		return typed
	}
	return &OAuthError{Code: "server_error", Description: err.Error(), StatusCode: 500}
}

type OAuthAuthorizationRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
}

type OAuthDynamicClientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

type OAuthDynamicClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope,omitempty"`
}

type OAuthAuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported,omitempty"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Resource     string `json:"resource,omitempty"`
}

type OAuthAuthorizationPrompt struct {
	ClientID    string
	ClientName  string
	Scope       string
	Resource    string
	RedirectURI string
}

type oauthPendingStateClaims struct {
	ReturnTo string `json:"return_to"`
	State    string `json:"state"`
	Use      string `json:"use"`
	jwt.RegisteredClaims
}

type oauthSessionClaims struct {
	Use string `json:"use"`
	jwt.RegisteredClaims
}

type signingKey struct {
	kid        string
	alg        string
	privateKey interface{}
	publicJWK  jwk
}

type oauthAuthorizationContext struct {
	client   *models.OAuthClient
	user     *models.User
	scope    string
	resource string
}

// OAuthServerService implements a self-hosted authorization server for MCP.
type OAuthServerService struct {
	mcpCfg             config.MCPConfig
	authCfg            config.AuthConfig
	userStore          *database.UserStore
	oauthStore         *database.OAuthStore
	googleCodeResolver func(context.Context, string, string) (*models.User, error)
	logger             *logging.Logger
	signer             *signingKey
	now                func() time.Time
	rand               io.Reader
}

func NewOAuthServerService(mcpCfg config.MCPConfig, authCfg config.AuthConfig, userStore *database.UserStore, oauthStore *database.OAuthStore, googleAuth *Service, logger *logging.Logger) *OAuthServerService {
	if !mcpCfg.Auth.Enabled || !mcpCfg.Auth.SelfHosted || userStore == nil || oauthStore == nil || googleAuth == nil {
		return nil
	}

	signer, err := loadOAuthSigningKey(mcpCfg.Auth, logger)
	if err != nil {
		logger.Error("Failed to initialize self-hosted OAuth signing key", logging.WithField("error", err.Error()))
		return nil
	}

	return &OAuthServerService{
		mcpCfg:     mcpCfg,
		authCfg:    authCfg,
		userStore:  userStore,
		oauthStore: oauthStore,
		googleCodeResolver: func(ctx context.Context, code, redirectURI string) (*models.User, error) {
			return googleAuth.resolveUserFromGoogleCode(ctx, code, redirectURI)
		},
		logger: logger,
		signer: signer,
		now:    time.Now,
		rand:   crand.Reader,
	}
}

func (s *OAuthServerService) Enabled() bool {
	return s != nil && s.signer != nil
}

func (s *OAuthServerService) SecureCookies() bool {
	issuer := strings.TrimSpace(s.mcpCfg.Auth.Issuer)
	return strings.HasPrefix(strings.ToLower(issuer), "https://")
}

func (s *OAuthServerService) SessionCookieName() string { return oauthSessionCookieName }
func (s *OAuthServerService) PendingCookieName() string { return oauthPendingCookieName }
func (s *OAuthServerService) SessionCookieTTL() time.Duration {
	return s.mcpCfg.Auth.SessionTTL
}
func (s *OAuthServerService) PendingCookieTTL() time.Duration {
	return oauthStateTTL
}

func (s *OAuthServerService) AuthorizationServerMetadata() *OAuthAuthorizationServerMetadata {
	if !s.Enabled() {
		return nil
	}
	issuer := strings.TrimRight(strings.TrimSpace(s.mcpCfg.Auth.Issuer), "/")
	return &OAuthAuthorizationServerMetadata{
		Issuer:                            issuer,
		AuthorizationEndpoint:             issuer + "/oauth/authorize",
		TokenEndpoint:                     issuer + "/oauth/token",
		RegistrationEndpoint:              issuer + "/oauth/register",
		JWKSURI:                           issuer + "/oauth/jwks.json",
		ScopesSupported:                   append([]string(nil), s.mcpCfg.Auth.RequiredScopes...),
		ResponseTypesSupported:            []string{models.OAuthResponseTypeCode},
		GrantTypesSupported:               []string{models.OAuthGrantTypeAuthorizationCode, models.OAuthGrantTypeRefreshToken},
		TokenEndpointAuthMethodsSupported: []string{models.OAuthTokenEndpointAuthMethodNone},
		CodeChallengeMethodsSupported:     []string{models.OAuthCodeChallengeMethodS256},
	}
}

func (s *OAuthServerService) JWKS() map[string]any {
	if !s.Enabled() {
		return nil
	}
	return map[string]any{
		"keys": []jwk{s.signer.publicJWK},
	}
}

func (s *OAuthServerService) RegisterClient(ctx context.Context, req OAuthDynamicClientRegistrationRequest) (*OAuthDynamicClientRegistrationResponse, error) {
	if !s.Enabled() {
		return nil, &OAuthError{Code: "server_error", Description: "self-hosted OAuth is not enabled", StatusCode: 503}
	}

	redirectURIs := uniqueStrings(req.RedirectURIs)
	if len(redirectURIs) == 0 {
		return nil, &OAuthError{Code: "invalid_client_metadata", Description: "redirect_uris must include at least one HTTPS URI", StatusCode: 400}
	}
	for _, redirectURI := range redirectURIs {
		parsed, err := url.Parse(redirectURI)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return nil, &OAuthError{Code: "invalid_client_metadata", Description: "redirect_uris must contain valid HTTPS URLs", StatusCode: 400}
		}
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{models.OAuthGrantTypeAuthorizationCode, models.OAuthGrantTypeRefreshToken}
	}
	if !sameStringSet(grantTypes, []string{models.OAuthGrantTypeAuthorizationCode, models.OAuthGrantTypeRefreshToken}) && !sameStringSet(grantTypes, []string{models.OAuthGrantTypeAuthorizationCode}) {
		return nil, &OAuthError{Code: "invalid_client_metadata", Description: "grant_types must only include authorization_code and optional refresh_token", StatusCode: 400}
	}

	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{models.OAuthResponseTypeCode}
	}
	if !sameStringSet(responseTypes, []string{models.OAuthResponseTypeCode}) {
		return nil, &OAuthError{Code: "invalid_client_metadata", Description: "response_types must only include code", StatusCode: 400}
	}

	tokenAuthMethod := strings.TrimSpace(req.TokenEndpointAuthMethod)
	if tokenAuthMethod == "" {
		tokenAuthMethod = models.OAuthTokenEndpointAuthMethodNone
	}
	if tokenAuthMethod != models.OAuthTokenEndpointAuthMethodNone {
		return nil, &OAuthError{Code: "invalid_client_metadata", Description: "only public OAuth clients without a client secret are supported", StatusCode: 400}
	}

	scope, err := s.normalizeRequestedScope(req.Scope)
	if err != nil {
		return nil, err
	}

	clientID, err := s.randomOpaqueToken(24)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to generate client id", StatusCode: 500}
	}
	clientID = "ff_mcp_" + clientID

	client, err := s.oauthStore.CreateClient(ctx, clientID, strings.TrimSpace(req.ClientName), redirectURIs, grantTypes, responseTypes, tokenAuthMethod, scope)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to register OAuth client", StatusCode: 500}
	}

	return &OAuthDynamicClientRegistrationResponse{
		ClientID:                client.ClientID,
		ClientIDIssuedAt:        client.CreatedAt.Unix(),
		ClientName:              client.ClientName,
		RedirectURIs:            client.RedirectURIs,
		GrantTypes:              client.GrantTypes,
		ResponseTypes:           client.ResponseTypes,
		TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
		Scope:                   client.Scope,
	}, nil
}

func (s *OAuthServerService) ParseAuthorizationRequest(values url.Values) (*OAuthAuthorizationRequest, error) {
	if !s.Enabled() {
		return nil, &OAuthError{Code: "server_error", Description: "self-hosted OAuth is not enabled", StatusCode: 503}
	}

	req := &OAuthAuthorizationRequest{
		ResponseType:        strings.TrimSpace(values.Get("response_type")),
		ClientID:            strings.TrimSpace(values.Get("client_id")),
		RedirectURI:         strings.TrimSpace(values.Get("redirect_uri")),
		Scope:               strings.TrimSpace(values.Get("scope")),
		State:               values.Get("state"),
		CodeChallenge:       strings.TrimSpace(values.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(values.Get("code_challenge_method")),
		Resource:            strings.TrimSpace(values.Get("resource")),
	}

	if req.ResponseType != models.OAuthResponseTypeCode {
		return nil, &OAuthError{Code: "unsupported_response_type", Description: "response_type must be code", StatusCode: 400}
	}
	if req.ClientID == "" || req.RedirectURI == "" {
		return nil, &OAuthError{Code: "invalid_request", Description: "client_id and redirect_uri are required", StatusCode: 400}
	}
	if req.CodeChallenge == "" || req.CodeChallengeMethod != models.OAuthCodeChallengeMethodS256 {
		return nil, &OAuthError{Code: "invalid_request", Description: "code_challenge and code_challenge_method=S256 are required", StatusCode: 400}
	}
	if _, err := s.normalizeRequestedScope(req.Scope); err != nil {
		return nil, err
	}
	if _, err := url.Parse(req.RedirectURI); err != nil {
		return nil, &OAuthError{Code: "invalid_request", Description: "redirect_uri must be a valid URL", StatusCode: 400}
	}
	if err := s.validateRequestedResource(req.Resource); err != nil {
		return nil, err
	}

	return req, nil
}

func (s *OAuthServerService) DescribeAuthorizationRequest(ctx context.Context, req *OAuthAuthorizationRequest, userID string) (*OAuthAuthorizationPrompt, error) {
	authCtx, err := s.validateAuthorizationRequest(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	clientName := strings.TrimSpace(authCtx.client.ClientName)
	if clientName == "" {
		clientName = "this app"
	}

	return &OAuthAuthorizationPrompt{
		ClientID:    authCtx.client.ClientID,
		ClientName:  clientName,
		Scope:       authCtx.scope,
		Resource:    authCtx.resource,
		RedirectURI: req.RedirectURI,
	}, nil
}

func (s *OAuthServerService) AuthorizationErrorRedirect(ctx context.Context, req *OAuthAuthorizationRequest, err error) (string, bool) {
	if req == nil || strings.TrimSpace(req.ClientID) == "" || strings.TrimSpace(req.RedirectURI) == "" {
		return "", false
	}

	client, lookupErr := s.oauthStore.GetClientByClientID(ctx, req.ClientID)
	if lookupErr != nil || client == nil || !containsString(client.RedirectURIs, req.RedirectURI) {
		return "", false
	}

	redirectURL, buildErr := buildAuthorizationErrorRedirect(req.RedirectURI, req.State, NormalizeOAuthError(err))
	if buildErr != nil {
		return "", false
	}
	return redirectURL, true
}

func (s *OAuthServerService) validateAuthorizationRequest(ctx context.Context, req *OAuthAuthorizationRequest, userID string) (*oauthAuthorizationContext, error) {
	if req == nil {
		return nil, &OAuthError{Code: "invalid_request", Description: "authorization request is required", StatusCode: 400}
	}
	if strings.TrimSpace(userID) == "" {
		return nil, &OAuthError{Code: "access_denied", Description: "user session is required", StatusCode: 401}
	}

	client, err := s.oauthStore.GetClientByClientID(ctx, req.ClientID)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load OAuth client", StatusCode: 500}
	}
	if client == nil {
		return nil, &OAuthError{Code: "unauthorized_client", Description: "unknown OAuth client", StatusCode: 400}
	}
	if !containsString(client.RedirectURIs, req.RedirectURI) {
		return nil, &OAuthError{Code: "invalid_request", Description: "redirect_uri is not registered for this client", StatusCode: 400}
	}
	if !containsString(client.ResponseTypes, models.OAuthResponseTypeCode) || !containsString(client.GrantTypes, models.OAuthGrantTypeAuthorizationCode) {
		return nil, &OAuthError{Code: "unauthorized_client", Description: "OAuth client is not allowed to use the authorization-code flow", StatusCode: 400}
	}

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load user", StatusCode: 500}
	}
	if user == nil || user.Status != models.UserStatusActive {
		return nil, &OAuthError{Code: "access_denied", Description: "user account is unavailable", StatusCode: 403}
	}

	scope, err := s.normalizeRequestedScope(req.Scope)
	if err != nil {
		return nil, err
	}
	if client.Scope != "" && scope != client.Scope {
		return nil, &OAuthError{Code: "invalid_scope", Description: "requested scope does not match the registered client scope", StatusCode: 400}
	}

	resource := strings.TrimSpace(req.Resource)
	if resource == "" {
		resource = strings.TrimSpace(s.mcpCfg.Auth.Resource)
	}

	return &oauthAuthorizationContext{
		client:   client,
		user:     user,
		scope:    scope,
		resource: resource,
	}, nil
}

func (s *OAuthServerService) Authorize(ctx context.Context, req *OAuthAuthorizationRequest, userID string) (string, error) {
	if !s.Enabled() {
		return "", &OAuthError{Code: "server_error", Description: "self-hosted OAuth is not enabled", StatusCode: 503}
	}
	authCtx, err := s.validateAuthorizationRequest(ctx, req, userID)
	if err != nil {
		return "", err
	}
	s.pruneExpiredOAuthState(ctx)

	codeValue, err := s.randomOpaqueToken(32)
	if err != nil {
		return "", &OAuthError{Code: "server_error", Description: "failed to generate authorization code", StatusCode: 500}
	}

	_, err = s.oauthStore.CreateAuthorizationCode(
		ctx,
		hashToken(codeValue),
		req.ClientID,
		authCtx.user.ID,
		req.RedirectURI,
		authCtx.scope,
		authCtx.resource,
		req.CodeChallenge,
		req.CodeChallengeMethod,
		s.now().Add(s.mcpCfg.Auth.AuthorizationCodeTTL),
	)
	if err != nil {
		return "", &OAuthError{Code: "server_error", Description: "failed to persist authorization code", StatusCode: 500}
	}

	redirectURL, err := url.Parse(req.RedirectURI)
	if err != nil {
		return "", &OAuthError{Code: "invalid_request", Description: "invalid redirect_uri", StatusCode: 400}
	}
	query := redirectURL.Query()
	query.Set("code", codeValue)
	if req.State != "" {
		query.Set("state", req.State)
	}
	redirectURL.RawQuery = query.Encode()
	return redirectURL.String(), nil
}

func (s *OAuthServerService) BuildGoogleAuthorizationURL(returnTo string) (string, string, error) {
	if !s.Enabled() {
		return "", "", &OAuthError{Code: "server_error", Description: "self-hosted OAuth is not enabled", StatusCode: 503}
	}
	if strings.TrimSpace(s.authCfg.GoogleClientID) == "" {
		return "", "", &OAuthError{Code: "server_error", Description: "Google OAuth client ID is not configured", StatusCode: 500}
	}
	if strings.TrimSpace(s.authCfg.GoogleClientSecret) == "" {
		return "", "", &OAuthError{Code: "server_error", Description: "Google OAuth client secret is not configured", StatusCode: 500}
	}

	googleState, err := s.randomOpaqueToken(24)
	if err != nil {
		return "", "", &OAuthError{Code: "server_error", Description: "failed to prepare login state", StatusCode: 500}
	}

	pendingToken, err := s.signPendingToken(returnTo, googleState)
	if err != nil {
		return "", "", &OAuthError{Code: "server_error", Description: "failed to encode pending login state", StatusCode: 500}
	}

	values := url.Values{}
	values.Set("client_id", s.authCfg.GoogleClientID)
	values.Set("redirect_uri", s.mcpCfg.Auth.GoogleRedirectURI)
	values.Set("response_type", "code")
	values.Set("scope", "openid email profile")
	values.Set("state", googleState)
	values.Set("access_type", "offline")
	values.Set("prompt", "select_account")

	return oauthGoogleAuthorizeURL + "?" + values.Encode(), pendingToken, nil
}

func (s *OAuthServerService) HandleGoogleCallback(ctx context.Context, code, state, pendingToken string) (string, string, error) {
	pending, err := s.parsePendingToken(pendingToken, state)
	if err != nil {
		return "", "", err
	}

	user, err := s.googleCodeResolver(ctx, code, s.mcpCfg.Auth.GoogleRedirectURI)
	if err != nil {
		if authErr, ok := err.(*AuthError); ok {
			return "", "", &OAuthError{Code: "access_denied", Description: authErr.Message, StatusCode: 401}
		}
		return "", "", &OAuthError{Code: "server_error", Description: "failed to resolve Google user", StatusCode: 500}
	}
	if user.Status != models.UserStatusActive {
		return "", "", &OAuthError{Code: "access_denied", Description: "user account is disabled", StatusCode: 403}
	}
	if err := s.userStore.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login for self-hosted OAuth session", logging.WithField("error", err.Error()))
	}

	sessionToken, err := s.signSessionToken(user.ID)
	if err != nil {
		return "", "", &OAuthError{Code: "server_error", Description: "failed to create login session", StatusCode: 500}
	}

	return sessionToken, pending.ReturnTo, nil
}

func (s *OAuthServerService) ValidateSessionToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", &OAuthError{Code: "access_denied", Description: "missing OAuth login session", StatusCode: 401}
	}

	parsedToken, err := jwt.ParseWithClaims(token, &oauthSessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.authCfg.JWTSecret), nil
	})
	if err != nil {
		return "", &OAuthError{Code: "access_denied", Description: "invalid OAuth login session", StatusCode: 401}
	}

	claims, ok := parsedToken.Claims.(*oauthSessionClaims)
	if !ok || !parsedToken.Valid || claims.Use != "mcp_oauth_session" {
		return "", &OAuthError{Code: "access_denied", Description: "invalid OAuth login session", StatusCode: 401}
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return "", &OAuthError{Code: "access_denied", Description: "invalid OAuth login session subject", StatusCode: 401}
	}

	return claims.Subject, nil
}

func (s *OAuthServerService) ExchangeToken(ctx context.Context, values url.Values) (*OAuthTokenResponse, error) {
	grantType := strings.TrimSpace(values.Get("grant_type"))
	switch grantType {
	case models.OAuthGrantTypeAuthorizationCode:
		return s.exchangeAuthorizationCode(ctx, values)
	case models.OAuthGrantTypeRefreshToken:
		return s.exchangeRefreshToken(ctx, values)
	default:
		return nil, &OAuthError{Code: "unsupported_grant_type", Description: "grant_type must be authorization_code or refresh_token", StatusCode: 400}
	}
}

func (s *OAuthServerService) exchangeAuthorizationCode(ctx context.Context, values url.Values) (*OAuthTokenResponse, error) {
	clientID := strings.TrimSpace(values.Get("client_id"))
	code := strings.TrimSpace(values.Get("code"))
	redirectURI := strings.TrimSpace(values.Get("redirect_uri"))
	codeVerifier := strings.TrimSpace(values.Get("code_verifier"))
	resource := strings.TrimSpace(values.Get("resource"))

	if clientID == "" || code == "" || redirectURI == "" || codeVerifier == "" {
		return nil, &OAuthError{Code: "invalid_request", Description: "client_id, code, redirect_uri, and code_verifier are required", StatusCode: 400}
	}

	client, err := s.oauthStore.GetClientByClientID(ctx, clientID)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load OAuth client", StatusCode: 500}
	}
	if client == nil {
		return nil, &OAuthError{Code: "invalid_client", Description: "unknown OAuth client", StatusCode: 401}
	}
	if !containsString(client.GrantTypes, models.OAuthGrantTypeAuthorizationCode) {
		return nil, &OAuthError{Code: "unauthorized_client", Description: "OAuth client is not allowed to use the authorization-code flow", StatusCode: 400}
	}

	authCode, err := s.oauthStore.GetAuthorizationCodeByHash(ctx, hashToken(code))
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load authorization code", StatusCode: 500}
	}
	if authCode == nil {
		return nil, &OAuthError{Code: "invalid_grant", Description: "authorization code is invalid, expired, or already used", StatusCode: 400}
	}
	if authCode.ClientID != clientID || authCode.RedirectURI != redirectURI {
		return nil, &OAuthError{Code: "invalid_grant", Description: "authorization code does not match this client or redirect URI", StatusCode: 400}
	}
	if resource != "" && authCode.Resource != "" && resource != authCode.Resource {
		return nil, &OAuthError{Code: "invalid_target", Description: "resource does not match the authorization grant", StatusCode: 400}
	}
	if !verifyS256Challenge(codeVerifier, authCode.CodeChallenge) {
		return nil, &OAuthError{Code: "invalid_grant", Description: "code_verifier does not match the authorization code challenge", StatusCode: 400}
	}
	consumed, err := s.oauthStore.MarkAuthorizationCodeConsumed(ctx, hashToken(code))
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to consume authorization code", StatusCode: 500}
	}
	if !consumed {
		return nil, &OAuthError{Code: "invalid_grant", Description: "authorization code is invalid, expired, or already used", StatusCode: 400}
	}

	return s.issueOAuthTokens(ctx, client, authCode.UserID, clientID, authCode.Scope, authCode.Resource)
}

func (s *OAuthServerService) exchangeRefreshToken(ctx context.Context, values url.Values) (*OAuthTokenResponse, error) {
	clientID := strings.TrimSpace(values.Get("client_id"))
	refreshToken := strings.TrimSpace(values.Get("refresh_token"))
	resource := strings.TrimSpace(values.Get("resource"))

	if clientID == "" || refreshToken == "" {
		return nil, &OAuthError{Code: "invalid_request", Description: "client_id and refresh_token are required", StatusCode: 400}
	}

	client, err := s.oauthStore.GetClientByClientID(ctx, clientID)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load OAuth client", StatusCode: 500}
	}
	if client == nil {
		return nil, &OAuthError{Code: "invalid_client", Description: "unknown OAuth client", StatusCode: 401}
	}
	if !containsString(client.GrantTypes, models.OAuthGrantTypeRefreshToken) {
		return nil, &OAuthError{Code: "unauthorized_client", Description: "OAuth client is not allowed to use the refresh-token flow", StatusCode: 400}
	}

	storedToken, err := s.oauthStore.GetRefreshTokenByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load refresh token", StatusCode: 500}
	}
	if storedToken == nil {
		return nil, &OAuthError{Code: "invalid_grant", Description: "refresh token is invalid, expired, or already used", StatusCode: 400}
	}
	if storedToken.ClientID != clientID {
		return nil, &OAuthError{Code: "invalid_grant", Description: "refresh token does not belong to this OAuth client", StatusCode: 400}
	}
	if resource != "" && storedToken.Resource != "" && resource != storedToken.Resource {
		return nil, &OAuthError{Code: "invalid_target", Description: "resource does not match the refresh token grant", StatusCode: 400}
	}
	revoked, err := s.oauthStore.MarkRefreshTokenRevoked(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to rotate refresh token", StatusCode: 500}
	}
	if !revoked {
		return nil, &OAuthError{Code: "invalid_grant", Description: "refresh token is invalid, expired, or already used", StatusCode: 400}
	}

	return s.issueOAuthTokens(ctx, client, storedToken.UserID, clientID, storedToken.Scope, storedToken.Resource)
}

func (s *OAuthServerService) issueOAuthTokens(ctx context.Context, client *models.OAuthClient, userID, clientID, scope, resource string) (*OAuthTokenResponse, error) {
	if client == nil {
		return nil, &OAuthError{Code: "server_error", Description: "OAuth client is required", StatusCode: 500}
	}

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, &OAuthError{Code: "server_error", Description: "failed to load user", StatusCode: 500}
	}
	if user == nil || user.Status != models.UserStatusActive {
		return nil, &OAuthError{Code: "invalid_grant", Description: "user account is unavailable", StatusCode: 400}
	}
	s.pruneExpiredOAuthState(ctx)

	accessToken, expiresIn, err := s.signAccessToken(user, clientID, scope, resource)
	if err != nil {
		return nil, err
	}

	response := &OAuthTokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       scope,
		Resource:    resource,
	}

	if containsString(client.GrantTypes, models.OAuthGrantTypeRefreshToken) {
		refreshValue, err := s.randomOpaqueToken(32)
		if err != nil {
			return nil, &OAuthError{Code: "server_error", Description: "failed to generate refresh token", StatusCode: 500}
		}
		_, err = s.oauthStore.CreateRefreshToken(ctx, user.ID, clientID, hashToken(refreshValue), scope, resource, s.now().Add(s.mcpCfg.Auth.RefreshTokenTTL))
		if err != nil {
			return nil, &OAuthError{Code: "server_error", Description: "failed to persist refresh token", StatusCode: 500}
		}
		response.RefreshToken = refreshValue
	}

	return response, nil
}

func (s *OAuthServerService) signAccessToken(user *models.User, clientID, scope, resource string) (string, int, error) {
	now := s.now()
	expiresAt := now.Add(s.mcpCfg.Auth.AccessTokenTTL)
	scopes := strings.Fields(scope)
	audience := accessTokenAudience(s.mcpCfg.Auth.Audience, resource)
	jti, err := s.randomOpaqueToken(24)
	if err != nil {
		return "", 0, &OAuthError{Code: "server_error", Description: "failed to generate access-token id", StatusCode: 500}
	}

	claims := jwt.MapClaims{
		"iss":            strings.TrimRight(strings.TrimSpace(s.mcpCfg.Auth.Issuer), "/"),
		"sub":            user.ID,
		"scope":          scope,
		"scp":            scopes,
		"email":          user.Email,
		"email_verified": true,
		"name":           user.EffectiveDisplayName(),
		"client_id":      clientID,
		"iat":            now.Unix(),
		"exp":            expiresAt.Unix(),
		"jti":            jti,
	}
	if audience != nil {
		claims["aud"] = audience
	}
	if resource != "" {
		claims["resource"] = resource
	}
	if picture := strings.TrimSpace(user.EffectiveAvatarURL()); picture != "" {
		claims["picture"] = picture
	}

	token := jwt.NewWithClaims(signingMethodForAlg(s.signer.alg), claims)
	token.Header["kid"] = s.signer.kid

	signed, err := token.SignedString(s.signer.privateKey)
	if err != nil {
		return "", 0, &OAuthError{Code: "server_error", Description: "failed to sign access token", StatusCode: 500}
	}

	return signed, int(s.mcpCfg.Auth.AccessTokenTTL.Seconds()), nil
}

func (s *OAuthServerService) signPendingToken(returnTo, googleState string) (string, error) {
	claims := oauthPendingStateClaims{
		ReturnTo: returnTo,
		State:    googleState,
		Use:      "mcp_oauth_pending",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(s.now().Add(oauthStateTTL)),
			IssuedAt:  jwt.NewNumericDate(s.now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.authCfg.JWTSecret))
}

func (s *OAuthServerService) parsePendingToken(tokenString, state string) (*oauthPendingStateClaims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &oauthPendingStateClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.authCfg.JWTSecret), nil
	})
	if err != nil {
		return nil, &OAuthError{Code: "access_denied", Description: "pending OAuth login state is invalid or expired", StatusCode: 401}
	}
	claims, ok := parsed.Claims.(*oauthPendingStateClaims)
	if !ok || !parsed.Valid || claims.Use != "mcp_oauth_pending" || claims.State != state || strings.TrimSpace(claims.ReturnTo) == "" {
		return nil, &OAuthError{Code: "access_denied", Description: "pending OAuth login state is invalid or expired", StatusCode: 401}
	}
	return claims, nil
}

func (s *OAuthServerService) signSessionToken(userID string) (string, error) {
	claims := oauthSessionClaims{
		Use: "mcp_oauth_session",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(s.now()),
			ExpiresAt: jwt.NewNumericDate(s.now().Add(s.mcpCfg.Auth.SessionTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.authCfg.JWTSecret))
}

func (s *OAuthServerService) pruneExpiredOAuthState(ctx context.Context) {
	if err := s.oauthStore.CleanupExpiredOAuthState(ctx); err != nil {
		s.logger.Warn("Failed to clean up expired self-hosted OAuth state", logging.WithField("error", err.Error()))
	}
}

func buildAuthorizationErrorRedirect(redirectURI, state string, oauthErr *OAuthError) (string, error) {
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}

	query := redirectURL.Query()
	query.Set("error", oauthErr.Code)
	if description := strings.TrimSpace(oauthErr.Description); description != "" {
		query.Set("error_description", description)
	}
	if state != "" {
		query.Set("state", state)
	}
	redirectURL.RawQuery = query.Encode()
	return redirectURL.String(), nil
}

func (s *OAuthServerService) normalizeRequestedScope(raw string) (string, error) {
	requested := strings.Fields(strings.TrimSpace(raw))
	if len(requested) == 0 {
		return strings.Join(s.mcpCfg.Auth.RequiredScopes, " "), nil
	}

	allowed := make(map[string]struct{}, len(s.mcpCfg.Auth.RequiredScopes))
	for _, scope := range s.mcpCfg.Auth.RequiredScopes {
		allowed[scope] = struct{}{}
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(requested))
	for _, scope := range requested {
		if _, ok := allowed[scope]; !ok {
			return "", &OAuthError{Code: "invalid_scope", Description: "requested scope is not supported", StatusCode: 400}
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}
	sort.Strings(normalized)
	return strings.Join(normalized, " "), nil
}

func (s *OAuthServerService) validateRequestedResource(resource string) error {
	resource = strings.TrimSpace(resource)
	if resource == "" || strings.TrimSpace(s.mcpCfg.Auth.Resource) == "" {
		return nil
	}
	if resource != strings.TrimSpace(s.mcpCfg.Auth.Resource) {
		return &OAuthError{Code: "invalid_target", Description: "resource must target this MCP server", StatusCode: 400}
	}
	return nil
}

func (s *OAuthServerService) randomOpaqueToken(byteLength int) (string, error) {
	buf := make([]byte, byteLength)
	if _, err := io.ReadFull(s.rand, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func verifyS256Challenge(codeVerifier, expectedChallenge string) bool {
	sum := sha256.Sum256([]byte(codeVerifier))
	actual := base64.RawURLEncoding.EncodeToString(sum[:])
	return actual == expectedChallenge
}

func accessTokenAudience(configuredAudience, resource string) interface{} {
	audiences := []string{}
	if configuredAudience = strings.TrimSpace(configuredAudience); configuredAudience != "" {
		audiences = append(audiences, configuredAudience)
	}
	if resource = strings.TrimSpace(resource); resource != "" && !containsString(audiences, resource) {
		audiences = append(audiences, resource)
	}
	switch len(audiences) {
	case 0:
		return nil
	case 1:
		return audiences[0]
	default:
		return audiences
	}
}

func signingMethodForAlg(alg string) jwt.SigningMethod {
	switch alg {
	case "RS256":
		return jwt.SigningMethodRS256
	case "ES256":
		return jwt.SigningMethodES256
	default:
		return jwt.SigningMethodES256
	}
}

func loadOAuthSigningKey(cfg config.MCPAuthConfig, logger *logging.Logger) (*signingKey, error) {
	if strings.TrimSpace(cfg.PrivateKeyPEM) == "" {
		if !cfg.AllowEphemeralKey {
			return nil, errors.New("MCP_AUTH_PRIVATE_KEY_PEM is required unless MCP_AUTH_ALLOW_EPHEMERAL_KEY is explicitly enabled")
		}
		logger.Warn("MCP self-hosted OAuth private key is not configured; generating an ephemeral ECDSA key for this process only")
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
		if err != nil {
			return nil, err
		}
		kid := strings.TrimSpace(cfg.KeyID)
		if kid == "" {
			kid = "ff-self-hosted-ephemeral"
		}
		publicJWK, err := publicJWKFromKey(kid, privateKey.Public())
		if err != nil {
			return nil, err
		}
		return &signingKey{kid: kid, alg: "ES256", privateKey: privateKey, publicJWK: publicJWK}, nil
	}

	block, _ := pem.Decode([]byte(cfg.PrivateKeyPEM))
	if block == nil {
		return nil, errors.New("invalid PEM block")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		switch typed := key.(type) {
		case *rsa.PrivateKey:
			kid := keyIDOrDefault(cfg.KeyID)
			publicJWK, err := publicJWKFromKey(kid, &typed.PublicKey)
			if err != nil {
				return nil, err
			}
			return &signingKey{kid: kid, alg: "RS256", privateKey: typed, publicJWK: publicJWK}, nil
		case *ecdsa.PrivateKey:
			kid := keyIDOrDefault(cfg.KeyID)
			publicJWK, err := publicJWKFromKey(kid, &typed.PublicKey)
			if err != nil {
				return nil, err
			}
			return &signingKey{kid: kid, alg: "ES256", privateKey: typed, publicJWK: publicJWK}, nil
		}
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		kid := keyIDOrDefault(cfg.KeyID)
		publicJWK, err := publicJWKFromKey(kid, &key.PublicKey)
		if err != nil {
			return nil, err
		}
		return &signingKey{kid: kid, alg: "ES256", privateKey: key, publicJWK: publicJWK}, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		kid := keyIDOrDefault(cfg.KeyID)
		publicJWK, err := publicJWKFromKey(kid, &key.PublicKey)
		if err != nil {
			return nil, err
		}
		return &signingKey{kid: kid, alg: "RS256", privateKey: key, publicJWK: publicJWK}, nil
	}

	return nil, errors.New("unsupported OAuth private key format")
}

func keyIDOrDefault(kid string) string {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return "ff-self-hosted"
	}
	return kid
}

func publicJWKFromKey(kid string, publicKey interface{}) (jwk, error) {
	switch key := publicKey.(type) {
	case *rsa.PublicKey:
		return jwk{
			Kty: "RSA",
			Kid: kid,
			Use: "sig",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(oauthBigEndianBytes(key.E)),
		}, nil
	case *ecdsa.PublicKey:
		curveName := ""
		switch key.Curve {
		case elliptic.P256():
			curveName = "P-256"
		default:
			return jwk{}, fmt.Errorf("unsupported ECDSA curve")
		}
		size := (key.Curve.Params().BitSize + 7) / 8
		return jwk{
			Kty: "EC",
			Kid: kid,
			Use: "sig",
			Alg: "ES256",
			Crv: curveName,
			X:   base64.RawURLEncoding.EncodeToString(paddedBytes(key.X.Bytes(), size)),
			Y:   base64.RawURLEncoding.EncodeToString(paddedBytes(key.Y.Bytes(), size)),
		}, nil
	default:
		return jwk{}, fmt.Errorf("unsupported public key type %T", publicKey)
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func sameStringSet(actual, expected []string) bool {
	actual = uniqueStrings(actual)
	expected = uniqueStrings(expected)
	if len(actual) != len(expected) {
		return false
	}
	sort.Strings(actual)
	sort.Strings(expected)
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

func oauthBigEndianBytes(n int) []byte {
	if n == 0 {
		return []byte{0}
	}
	bytes := []byte{}
	for n > 0 {
		bytes = append([]byte{byte(n & 0xff)}, bytes...)
		n >>= 8
	}
	return bytes
}

func paddedBytes(input []byte, size int) []byte {
	if len(input) >= size {
		return input
	}
	out := make([]byte, size)
	copy(out[size-len(input):], input)
	return out
}
