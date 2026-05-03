package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	defaultMCPReadScope = "flyingforge.read"
	jwksCacheTTL        = 10 * time.Minute
)

// MCPProtectedResourceMetadata is served at /.well-known/oauth-protected-resource
// so ChatGPT can discover how to authorize against private MCP tools.
type MCPProtectedResourceMetadata struct {
	Resource              string   `json:"resource"`
	AuthorizationServers  []string `json:"authorization_servers"`
	ScopesSupported       []string `json:"scopes_supported,omitempty"`
	ResourceDocumentation string   `json:"resource_documentation,omitempty"`
}

// MCPAuthError describes an MCP OAuth failure in a way the transport can turn
// into WWW-Authenticate challenges and tool-level auth prompts.
type MCPAuthError struct {
	Code    string
	Message string
	Scope   string
}

func (e *MCPAuthError) Error() string {
	return e.Message
}

func (e *MCPAuthError) ErrorCode() string {
	return e.Code
}

type oidcDiscoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type cachedSigningKeys struct {
	keys      map[string]interface{}
	fetchedAt time.Time
}

type mcpIdentity struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	Picture       string
}

// MCPAuthService verifies managed-provider access tokens and maps them to
// FlyingForge users for private MCP tool access.
type MCPAuthService struct {
	cfg       config.MCPConfig
	userStore *database.UserStore
	logger    *logging.Logger
	client    *http.Client

	mu            sync.Mutex
	discovery     *oidcDiscoveryDocument
	discoveryTime time.Time
	signingKeys   cachedSigningKeys
}

func NewMCPAuthService(cfg config.MCPConfig, userStore *database.UserStore, logger *logging.Logger) *MCPAuthService {
	if !cfg.Auth.Enabled || userStore == nil {
		return nil
	}

	return &MCPAuthService{
		cfg:       cfg,
		userStore: userStore,
		logger:    logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *MCPAuthService) Enabled() bool {
	return s != nil && s.cfg.Auth.Enabled
}

func (s *MCPAuthService) ResourceMetadataURL() string {
	if s == nil || strings.TrimSpace(s.cfg.PublicBaseURL) == "" {
		return ""
	}
	return strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/.well-known/oauth-protected-resource"
}

func (s *MCPAuthService) RequiredScopes() []string {
	if s == nil {
		return []string{defaultMCPReadScope}
	}
	if len(s.cfg.Auth.RequiredScopes) == 0 {
		return []string{defaultMCPReadScope}
	}
	return append([]string(nil), s.cfg.Auth.RequiredScopes...)
}

func (s *MCPAuthService) Challenge(errorCode, description string) string {
	scope := strings.Join(s.RequiredScopes(), " ")
	resourceMetadataURL := s.ResourceMetadataURL()

	params := []string{}
	if resourceMetadataURL != "" {
		params = append(params, fmt.Sprintf(`resource_metadata="%s"`, resourceMetadataURL))
	}
	if scope != "" {
		params = append(params, fmt.Sprintf(`scope="%s"`, scope))
	}
	if errorCode != "" {
		params = append(params, fmt.Sprintf(`error="%s"`, errorCode))
	}
	if description != "" {
		safeDescription := strings.ReplaceAll(description, `"`, `'`)
		params = append(params, fmt.Sprintf(`error_description="%s"`, safeDescription))
	}
	if len(params) == 0 {
		return "Bearer"
	}

	return "Bearer " + strings.Join(params, ", ")
}

func (s *MCPAuthService) ProtectedResourceMetadata() *MCPProtectedResourceMetadata {
	if !s.Enabled() {
		return nil
	}

	resource := strings.TrimSpace(s.cfg.Auth.Resource)
	if resource == "" {
		baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.PublicBaseURL), "/")
		if baseURL != "" {
			resource = baseURL + "/mcp"
		}
	}

	resourceDocumentation := ""
	if base := strings.TrimRight(strings.TrimSpace(s.cfg.PublicBaseURL), "/"); base != "" {
		resourceDocumentation = base
	}

	return &MCPProtectedResourceMetadata{
		Resource:              resource,
		AuthorizationServers:  []string{strings.TrimRight(strings.TrimSpace(s.cfg.Auth.Issuer), "/")},
		ScopesSupported:       s.RequiredScopes(),
		ResourceDocumentation: resourceDocumentation,
	}
}

func (s *MCPAuthService) AuthenticateBearerToken(ctx context.Context, token string) (string, error) {
	if !s.Enabled() {
		return "", &MCPAuthError{
			Code:    "invalid_token",
			Message: "MCP authentication is not configured",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication required: no access token provided.",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	claims, err := s.verifyToken(ctx, token)
	if err != nil {
		return "", err
	}

	identity, err := extractMCPIdentity(claims)
	if err != nil {
		return "", err
	}

	userID, err := s.resolveUser(ctx, identity)
	if err != nil {
		return "", err
	}

	if err := s.userStore.UpdateLastLogin(ctx, userID); err != nil {
		s.logger.Warn("Failed to update MCP user last login", logging.WithField("error", err.Error()))
	}

	return userID, nil
}

func (s *MCPAuthService) verifyToken(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	signingKeys, err := s.getSigningKeys(ctx)
	if err != nil {
		return nil, &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed while loading signing keys",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		switch token.Method.(type) {
		case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, _ := token.Header["kid"].(string)
		if kid != "" {
			key, ok := signingKeys[kid]
			if !ok {
				return nil, fmt.Errorf("signing key %q not found", kid)
			}
			return key, nil
		}

		if len(signingKeys) == 1 {
			for _, key := range signingKeys {
				return key, nil
			}
		}

		return nil, fmt.Errorf("missing signing key id")
	})
	if err != nil {
		return nil, &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: invalid or expired access token",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: invalid token claims",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	expectedIssuer := strings.TrimRight(strings.TrimSpace(s.cfg.Auth.Issuer), "/")
	if issuer, _ := claims["iss"].(string); strings.TrimRight(issuer, "/") != expectedIssuer {
		return nil, &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: token issuer does not match the configured MCP issuer",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	if !matchesExpectedAudienceOrResource(claims, s.cfg.Auth.Audience, s.cfg.Auth.Resource) {
		return nil, &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: token audience/resource does not match this MCP server",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	if !hasRequiredScopes(claims, s.RequiredScopes()) {
		return nil, &MCPAuthError{
			Code:    "insufficient_scope",
			Message: "Authentication failed: token is missing the required FlyingForge read scope",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	return claims, nil
}

func (s *MCPAuthService) getSigningKeys(ctx context.Context) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.signingKeys.keys) > 0 && time.Since(s.signingKeys.fetchedAt) < jwksCacheTTL {
		return s.signingKeys.keys, nil
	}

	discovery, err := s.getDiscoveryDocumentLocked(ctx)
	if err != nil {
		return nil, err
	}

	jwksURL := strings.TrimSpace(s.cfg.Auth.JWKSURL)
	if jwksURL == "" && discovery != nil {
		jwksURL = discovery.JWKSURI
	}
	if jwksURL == "" {
		return nil, fmt.Errorf("missing JWKS URL")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected JWKS status: %s", response.Status)
	}

	var keySet jwkSet
	if err := json.NewDecoder(response.Body).Decode(&keySet); err != nil {
		return nil, err
	}

	keys := make(map[string]interface{}, len(keySet.Keys))
	for _, key := range keySet.Keys {
		publicKey, err := key.publicKey()
		if err != nil {
			s.logger.Warn("Skipping unsupported JWKS entry", logging.WithField("error", err.Error()))
			continue
		}

		keyID := key.Kid
		if keyID == "" {
			keyID = fmt.Sprintf("%s:%s", key.Kty, key.Alg)
		}
		keys[keyID] = publicKey
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no usable signing keys found")
	}

	s.signingKeys = cachedSigningKeys{
		keys:      keys,
		fetchedAt: time.Now(),
	}

	return keys, nil
}

func (s *MCPAuthService) getDiscoveryDocumentLocked(ctx context.Context) (*oidcDiscoveryDocument, error) {
	if s.discovery != nil && time.Since(s.discoveryTime) < jwksCacheTTL {
		return s.discovery, nil
	}

	urls := []string{}
	if configured := strings.TrimSpace(s.cfg.Auth.DiscoveryURL); configured != "" {
		urls = append(urls, configured)
	} else {
		issuer := strings.TrimRight(strings.TrimSpace(s.cfg.Auth.Issuer), "/")
		if issuer != "" {
			urls = append(urls,
				issuer+"/.well-known/openid-configuration",
				issuer+"/.well-known/oauth-authorization-server",
			)
		}
	}

	var lastErr error
	for _, candidate := range urls {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		if err != nil {
			lastErr = err
			continue
		}
		request.Header.Set("Accept", "application/json")

		response, err := s.client.Do(request)
		if err != nil {
			lastErr = err
			continue
		}

		var doc oidcDiscoveryDocument
		if response.StatusCode == http.StatusOK {
			err = json.NewDecoder(response.Body).Decode(&doc)
			response.Body.Close()
			if err == nil && doc.JWKSURI != "" {
				s.discovery = &doc
				s.discoveryTime = time.Now()
				return s.discovery, nil
			}
			lastErr = err
			continue
		}

		lastErr = fmt.Errorf("unexpected discovery status: %s", response.Status)
		response.Body.Close()
	}

	if strings.TrimSpace(s.cfg.Auth.JWKSURL) != "" {
		return &oidcDiscoveryDocument{
			Issuer:  strings.TrimRight(strings.TrimSpace(s.cfg.Auth.Issuer), "/"),
			JWKSURI: strings.TrimSpace(s.cfg.Auth.JWKSURL),
		}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("OIDC discovery failed")
	}
	return nil, lastErr
}

func (s *MCPAuthService) resolveUser(ctx context.Context, identity *mcpIdentity) (string, error) {
	subjectKey := identity.Issuer + "|" + identity.Subject

	linkedIdentity, err := s.userStore.GetIdentityByProvider(ctx, models.AuthProviderMCPOAuth, subjectKey)
	if err != nil {
		return "", err
	}
	if linkedIdentity != nil {
		user, err := s.userStore.GetByID(ctx, linkedIdentity.UserID)
		if err != nil {
			return "", err
		}
		if user == nil || user.Status != models.UserStatusActive {
			return "", &MCPAuthError{
				Code:    "invalid_token",
				Message: "Authentication failed: linked FlyingForge account is unavailable",
				Scope:   strings.Join(s.RequiredScopes(), " "),
			}
		}
		return user.ID, nil
	}

	if !identity.EmailVerified || strings.TrimSpace(identity.Email) == "" {
		return "", &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: a verified email is required to link your FlyingForge account",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	email := strings.ToLower(strings.TrimSpace(identity.Email))
	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if user == nil {
		user, err = s.userStore.Create(ctx, models.CreateUserParams{
			Email:       email,
			DisplayName: "",
			AvatarURL:   identity.Picture,
			Status:      models.UserStatusActive,
		})
		if err != nil {
			return "", err
		}
	}

	if user.Status != models.UserStatusActive {
		return "", &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: your FlyingForge account is disabled",
			Scope:   strings.Join(s.RequiredScopes(), " "),
		}
	}

	if _, err := s.userStore.CreateIdentity(ctx, user.ID, models.AuthProviderMCPOAuth, subjectKey, email); err != nil && !strings.Contains(err.Error(), "identity already linked") {
		return "", err
	}

	return user.ID, nil
}

func extractMCPIdentity(claims jwt.MapClaims) (*mcpIdentity, error) {
	issuer, _ := claims["iss"].(string)
	subject, _ := claims["sub"].(string)
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(subject) == "" {
		return nil, &MCPAuthError{
			Code:    "invalid_token",
			Message: "Authentication failed: token is missing issuer or subject",
			Scope:   defaultMCPReadScope,
		}
	}

	email, _ := claims["email"].(string)
	picture, _ := claims["picture"].(string)

	return &mcpIdentity{
		Issuer:        strings.TrimRight(strings.TrimSpace(issuer), "/"),
		Subject:       strings.TrimSpace(subject),
		Email:         strings.TrimSpace(email),
		EmailVerified: claimBool(claims["email_verified"]),
		Picture:       strings.TrimSpace(picture),
	}, nil
}

func matchesExpectedAudienceOrResource(claims jwt.MapClaims, expectedAudience, expectedResource string) bool {
	targets := []string{}
	if expectedAudience = strings.TrimSpace(expectedAudience); expectedAudience != "" {
		targets = append(targets, expectedAudience)
	}
	if expectedResource = strings.TrimSpace(expectedResource); expectedResource != "" {
		targets = append(targets, expectedResource)
	}
	if len(targets) == 0 {
		return true
	}

	audiences := claimStrings(claims["aud"])
	resources := claimStrings(claims["resource"])

	for _, target := range targets {
		if containsString(audiences, target) || containsString(resources, target) {
			return true
		}
	}

	return false
}

func hasRequiredScopes(claims jwt.MapClaims, required []string) bool {
	if len(required) == 0 {
		return true
	}

	scopeValues := map[string]struct{}{}
	for _, scope := range strings.Fields(strings.TrimSpace(claimString(claims["scope"]))) {
		scopeValues[scope] = struct{}{}
	}
	for _, scope := range claimStrings(claims["scp"]) {
		scopeValues[scope] = struct{}{}
	}

	for _, requiredScope := range required {
		if _, ok := scopeValues[requiredScope]; !ok {
			return false
		}
	}

	return true
}

func claimBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func claimString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func claimStrings(value interface{}) []string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				result = append(result, strings.TrimSpace(str))
			}
		}
		return result
	default:
		return nil
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (j jwk) publicKey() (interface{}, error) {
	switch strings.ToUpper(j.Kty) {
	case "RSA":
		return rsaPublicKeyFromJWK(j)
	case "EC":
		return ecdsaPublicKeyFromJWK(j)
	default:
		return nil, fmt.Errorf("unsupported JWK type %q", j.Kty)
	}
}

func rsaPublicKeyFromJWK(key jwk) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA modulus: %w", err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA exponent: %w", err)
	}

	exponent := 0
	for _, b := range exponentBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, fmt.Errorf("invalid RSA exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulusBytes),
		E: exponent,
	}, nil
}

func ecdsaPublicKeyFromJWK(key jwk) (*ecdsa.PublicKey, error) {
	curve, err := curveFromName(key.Crv)
	if err != nil {
		return nil, err
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, fmt.Errorf("invalid EC x coordinate: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid EC y coordinate: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func curveFromName(name string) (elliptic.Curve, error) {
	switch name {
	case "P-256":
		return elliptic.P256(), nil
	case "P-384":
		return elliptic.P384(), nil
	case "P-521":
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("unsupported EC curve %q", name)
	}
}
