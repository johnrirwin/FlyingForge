package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
)

const oauthCORSDefaultAllowedHeaders = "Authorization, Content-Type, Accept, Last-Event-ID, MCP-Session-Id"

var authorizeConsentTemplate = template.Must(template.New("oauth-authorize-consent").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Authorize {{.Prompt.ClientName}}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0b1020; color: #e5e7eb; margin: 0; padding: 2rem; }
    main { max-width: 34rem; margin: 0 auto; background: #111827; border-radius: 16px; padding: 2rem; box-shadow: 0 16px 48px rgba(0,0,0,0.28); }
    h1 { margin-top: 0; font-size: 1.6rem; }
    p, li { line-height: 1.5; }
    ul { padding-left: 1.2rem; }
    .actions { display: flex; gap: 0.75rem; margin-top: 1.5rem; }
    button { border: 0; border-radius: 999px; padding: 0.85rem 1.2rem; font-size: 1rem; cursor: pointer; }
    .approve { background: #2563eb; color: white; }
    .deny { background: #374151; color: #f3f4f6; }
    .meta { color: #9ca3af; font-size: 0.95rem; }
    .app-name { color: #93c5fd; }
  </style>
</head>
  <body>
  <main>
    <h1>Allow <span class="app-name">{{.Prompt.ClientName}}</span> to access FlyingForge?</h1>
    <p>{{.Prompt.ClientName}} is requesting access to your FlyingForge account.</p>
    <ul>
      {{range .AccessDescriptions}}<li>{{.}}</li>{{end}}
    </ul>
    <p class="meta">Review the requested permissions above before approving this connection.</p>
    <form method="post" action="/oauth/authorize">
      <input type="hidden" name="response_type" value="{{.Request.ResponseType}}">
      <input type="hidden" name="client_id" value="{{.Request.ClientID}}">
      <input type="hidden" name="redirect_uri" value="{{.Request.RedirectURI}}">
      <input type="hidden" name="scope" value="{{.Request.Scope}}">
      <input type="hidden" name="state" value="{{.Request.State}}">
      <input type="hidden" name="response_mode" value="{{.Request.ResponseMode}}">
      <input type="hidden" name="code_challenge" value="{{.Request.CodeChallenge}}">
      <input type="hidden" name="code_challenge_method" value="{{.Request.CodeChallengeMethod}}">
      <input type="hidden" name="resource" value="{{.Request.Resource}}">
      <input type="hidden" name="consent_token" value="{{.ConsentToken}}">
      <div class="actions">
        <button class="approve" type="submit" name="decision" value="approve">Approve</button>
        <button class="deny" type="submit" name="decision" value="deny">Deny</button>
      </div>
    </form>
  </main>
</body>
</html>`))

var authorizeWebMessageTemplate = template.Must(template.New("oauth-authorize-web-message").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Authorization complete</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0b1020; color: #e5e7eb; margin: 0; padding: 2rem; display: flex; align-items: center; justify-content: center; min-height: 100vh; }
    main { max-width: 34rem; background: #111827; border-radius: 16px; padding: 2rem; box-shadow: 0 16px 48px rgba(0,0,0,0.28); }
    h1 { margin-top: 0; font-size: 1.5rem; }
    p { line-height: 1.5; }
    .meta { color: #9ca3af; font-size: 0.95rem; }
  </style>
</head>
<body>
  <main>
    <h1>{{.Title}}</h1>
    <p>{{.Message}}</p>
    <p class="meta">If this window does not close automatically, you can close it and return to your app.</p>
  </main>
  <script>
    (function () {
      const payload = {{.PayloadJSON}};
      const targetOrigin = {{.TargetOriginJSON}};
      const responseMode = {{.ResponseModeJSON}};

      let targetWindow = null;
      if (responseMode === "web_message.opener") {
        targetWindow = window.opener;
      } else if (window.opener && !window.opener.closed) {
        targetWindow = window.opener;
      } else if (window.parent && window.parent !== window) {
        targetWindow = window.parent;
      }

      if (!targetWindow || !targetOrigin) {
        return;
      }

      targetWindow.postMessage(payload, targetOrigin);
      window.close();
    })();
  </script>
</body>
</html>`))

// OAuthAPI exposes a self-hosted authorization server for MCP.
type OAuthAPI struct {
	oauthService   *auth.OAuthServerService
	allowedOrigins map[string]struct{}
	logger         *logging.Logger
}

func NewOAuthAPI(oauthService *auth.OAuthServerService, logger *logging.Logger) *OAuthAPI {
	originSet := map[string]struct{}{}
	if oauthService != nil {
		for _, origin := range oauthService.AllowedOrigins() {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				originSet[trimmed] = struct{}{}
			}
		}
	}

	return &OAuthAPI{
		oauthService:   oauthService,
		allowedOrigins: originSet,
		logger:         logger,
	}
}

func (api *OAuthAPI) RegisterRoutes(mux *http.ServeMux) {
	if api == nil || api.oauthService == nil || !api.oauthService.Enabled() {
		return
	}
	mux.HandleFunc("/.well-known/openid-configuration", api.handleOpenIDConfiguration)
	mux.HandleFunc("/.well-known/oauth-authorization-server", api.handleAuthorizationServerMetadata)
	mux.HandleFunc("/oauth/jwks.json", api.handleJWKS)
	mux.HandleFunc("/oauth/register", api.handleRegisterClient)
	mux.HandleFunc("/oauth/authorize", api.handleAuthorize)
	mux.HandleFunc("/oauth/token", api.handleToken)
	mux.HandleFunc("/oauth/google/callback", api.handleGoogleCallback)
}

func (api *OAuthAPI) handleOpenIDConfiguration(w http.ResponseWriter, r *http.Request) {
	api.handleMetadataResponse(w, r)
}

func (api *OAuthAPI) handleAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	api.handleMetadataResponse(w, r)
}

func (api *OAuthAPI) handleMetadataResponse(w http.ResponseWriter, r *http.Request) {
	if api.handleCORS(w, r, "GET, OPTIONS") {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeOAuthJSON(w, http.StatusOK, api.oauthService.AuthorizationServerMetadata())
}

func (api *OAuthAPI) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if api.handleCORS(w, r, "GET, OPTIONS") {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeOAuthJSON(w, http.StatusOK, api.oauthService.JWKS())
}

func (api *OAuthAPI) handleRegisterClient(w http.ResponseWriter, r *http.Request) {
	if api.handleCORS(w, r, "POST, OPTIONS") {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req auth.OAuthDynamicClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeOAuthError(w, &auth.OAuthError{Code: "invalid_client_metadata", Description: "invalid registration payload", StatusCode: http.StatusBadRequest})
		return
	}

	response, err := api.oauthService.RegisterClient(r.Context(), req)
	if err != nil {
		api.writeOAuthError(w, err)
		return
	}

	writeOAuthJSON(w, http.StatusCreated, response)
}

func (api *OAuthAPI) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if api.handleCORS(w, r, "GET, POST, OPTIONS") {
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var values url.Values
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			api.writeOAuthError(w, &auth.OAuthError{Code: "invalid_request", Description: "invalid authorization form body", StatusCode: http.StatusBadRequest})
			return
		}
		values = r.PostForm
	} else {
		values = r.URL.Query()
	}

	authReq, err := api.oauthService.ParseAuthorizationRequest(values)
	if err != nil {
		api.writeOAuthError(w, err)
		return
	}

	userID := ""
	if sessionCookie, err := r.Cookie(api.oauthService.SessionCookieName()); err == nil {
		validatedUserID, sessionErr := api.oauthService.ValidateSessionToken(sessionCookie.Value)
		if sessionErr == nil {
			userID = validatedUserID
		} else {
			if api.logger != nil {
				api.logger.Warn("Invalid self-hosted OAuth session cookie", logging.WithFields(map[string]interface{}{
					"method": r.Method,
					"error":  sessionErr.Error(),
				}))
			}
			api.clearCookie(w, api.oauthService.SessionCookieName())
		}
	}

	if r.Method == http.MethodGet {
		if userID != "" {
			prompt, promptErr := api.oauthService.DescribeAuthorizationRequest(r.Context(), authReq, userID)
			if promptErr != nil {
				api.redirectOrWriteAuthorizeError(w, r, authReq, promptErr)
				return
			}
			api.renderAuthorizeConsentPage(w, authReq, prompt)
			return
		}

		googleURL, pendingToken, err := api.oauthService.BuildGoogleAuthorizationURL(r.URL.RequestURI())
		if err != nil {
			api.writeOAuthError(w, err)
			return
		}
		api.setCookie(w, api.oauthService.PendingCookieName(), pendingToken, api.oauthService.PendingCookieTTL())
		http.Redirect(w, r, googleURL, http.StatusFound)
		return
	}

	if userID == "" {
		if api.logger != nil {
			api.logger.Warn("OAuth consent submission missing valid session; restarting authorization", logging.WithFields(map[string]interface{}{
				"client_id": authReq.ClientID,
			}))
		}
		api.clearCookie(w, api.oauthService.SessionCookieName())
		redirectURL := "/oauth/authorize?" + r.PostForm.Encode()
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	switch strings.ToLower(strings.TrimSpace(r.PostForm.Get("decision"))) {
	case "approve":
		if consentErr := api.oauthService.ValidateAuthorizationConsentToken(r.PostForm.Get("consent_token"), userID, authReq); consentErr != nil {
			api.logOAuthAuthorizeFailure(r, authReq, consentErr)
			api.redirectOrWriteAuthorizeError(w, r, authReq, consentErr)
			return
		}
		redirectURL, authorizeErr := api.oauthService.Authorize(r.Context(), authReq, userID)
		if authorizeErr != nil {
			api.logOAuthAuthorizeFailure(r, authReq, authorizeErr)
			api.redirectOrWriteAuthorizeError(w, r, authReq, authorizeErr)
			return
		}
		api.logOAuthAuthorizeSuccess(r, authReq, redirectURL)
		if api.writeAuthorizeWebMessageResponse(w, authReq, redirectURL, nil) {
			return
		}
		http.Redirect(w, r, redirectURL, redirectStatusCode(r))
	case "deny":
		denyErr := &auth.OAuthError{
			Code:        "access_denied",
			Description: "user denied the authorization request",
			StatusCode:  http.StatusUnauthorized,
		}
		api.logOAuthAuthorizeFailure(r, authReq, denyErr)
		api.redirectOrWriteAuthorizeError(w, r, authReq, denyErr)
	default:
		decisionErr := &auth.OAuthError{
			Code:        "invalid_request",
			Description: "authorization decision is required",
			StatusCode:  http.StatusBadRequest,
		}
		api.logOAuthAuthorizeFailure(r, authReq, decisionErr)
		api.redirectOrWriteAuthorizeError(w, r, authReq, decisionErr)
	}
}

func (api *OAuthAPI) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if api.handleCORS(w, r, "GET, OPTIONS") {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if authError := r.URL.Query().Get("error"); authError != "" {
		api.clearCookie(w, api.oauthService.PendingCookieName())
		oauthErr := &auth.OAuthError{Code: "access_denied", Description: "Google authentication was denied", StatusCode: http.StatusUnauthorized}
		api.logOAuthCallbackFailure(r, oauthErr)
		api.writeOAuthError(w, oauthErr)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	pendingCookie, err := r.Cookie(api.oauthService.PendingCookieName())
	if err != nil || code == "" || state == "" {
		oauthErr := &auth.OAuthError{Code: "access_denied", Description: "missing OAuth login callback state", StatusCode: http.StatusUnauthorized}
		api.logOAuthCallbackFailure(r, oauthErr)
		api.writeOAuthError(w, oauthErr)
		return
	}

	sessionToken, redirectTo, callbackErr := api.oauthService.HandleGoogleCallback(r.Context(), code, state, pendingCookie.Value)
	if callbackErr != nil {
		api.clearCookie(w, api.oauthService.PendingCookieName())
		api.logOAuthCallbackFailure(r, callbackErr)
		api.writeOAuthError(w, callbackErr)
		return
	}

	api.clearCookie(w, api.oauthService.PendingCookieName())
	api.setCookie(w, api.oauthService.SessionCookieName(), sessionToken, api.oauthService.SessionCookieTTL())
	api.logOAuthCallbackSuccess(r, redirectTo)
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (api *OAuthAPI) handleToken(w http.ResponseWriter, r *http.Request) {
	if api.handleCORS(w, r, "POST, OPTIONS") {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		oauthErr := &auth.OAuthError{Code: "invalid_request", Description: "invalid token request body", StatusCode: http.StatusBadRequest}
		api.logOAuthTokenFailure(r, nil, oauthErr)
		api.writeOAuthError(w, oauthErr)
		return
	}

	response, err := api.oauthService.ExchangeToken(r.Context(), r.PostForm)
	if err != nil {
		api.logOAuthTokenFailure(r, r.PostForm, err)
		api.writeOAuthError(w, err)
		return
	}
	api.logOAuthTokenSuccess(r, r.PostForm)
	writeOAuthJSON(w, http.StatusOK, response)
}

func (api *OAuthAPI) writeOAuthError(w http.ResponseWriter, err error) {
	oauthErr := auth.NormalizeOAuthError(err)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(oauthErr.StatusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             oauthErr.Code,
		"error_description": oauthErr.Description,
	})
}

func (api *OAuthAPI) redirectOrWriteAuthorizeError(w http.ResponseWriter, r *http.Request, authReq *auth.OAuthAuthorizationRequest, err error) {
	if redirectURL, ok := api.oauthService.AuthorizationErrorRedirect(r.Context(), authReq, err); ok {
		if api.writeAuthorizeWebMessageResponse(w, authReq, "", err) {
			return
		}
		http.Redirect(w, r, redirectURL, redirectStatusCode(r))
		return
	}
	api.writeOAuthError(w, err)
}

func (api *OAuthAPI) renderAuthorizeConsentPage(w http.ResponseWriter, authReq *auth.OAuthAuthorizationRequest, prompt *auth.OAuthAuthorizationPrompt) {
	consentToken, err := api.oauthService.BuildAuthorizationConsentToken(prompt.UserID, authReq)
	if err != nil {
		http.Error(w, "failed to prepare authorization prompt", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	if err := authorizeConsentTemplate.Execute(w, map[string]any{
		"Prompt":             prompt,
		"Request":            authReq,
		"ConsentToken":       consentToken,
		"AccessDescriptions": describeAuthorizationAccess(prompt.Scope),
	}); err != nil {
		http.Error(w, "failed to render authorization prompt", http.StatusInternalServerError)
	}
}

func describeAuthorizationAccess(scope string) []string {
	scopeSet := map[string]struct{}{}
	for _, value := range strings.Fields(strings.TrimSpace(scope)) {
		scopeSet[value] = struct{}{}
	}

	descriptions := make([]string, 0, len(scopeSet))
	if _, ok := scopeSet["flyingforge.read"]; ok {
		descriptions = append(descriptions, "View your aircraft, receiver summaries, tuning, radios, and backup metadata.")
		descriptions = append(descriptions, "Use read-only access only; this app cannot modify your FlyingForge data.")
		delete(scopeSet, "flyingforge.read")
	}

	remainingScopes := make([]string, 0, len(scopeSet))
	for value := range scopeSet {
		remainingScopes = append(remainingScopes, value)
	}
	sort.Strings(remainingScopes)

	for _, value := range remainingScopes {
		descriptions = append(descriptions, "Access scope: "+value)
	}
	if len(descriptions) == 0 {
		descriptions = append(descriptions, "Read the data you approve for this FlyingForge connection.")
	}
	return descriptions
}

func redirectStatusCode(r *http.Request) int {
	if r != nil && r.Method == http.MethodPost {
		return http.StatusSeeOther
	}
	return http.StatusFound
}

func (api *OAuthAPI) writeAuthorizeWebMessageResponse(w http.ResponseWriter, authReq *auth.OAuthAuthorizationRequest, redirectURL string, err error) bool {
	if !usesWebMessageResponseMode(authReq) {
		return false
	}
	targetOrigin := originFromRedirectURI(authReq.RedirectURI)
	if targetOrigin == "" {
		return false
	}

	payload := map[string]string{
		"iss": strings.TrimSpace(api.oauthService.AuthorizationServerMetadata().Issuer),
	}
	title := "Authorization complete"
	message := "You can return to your app now."

	if err != nil {
		oauthErr := auth.NormalizeOAuthError(err)
		payload["error"] = oauthErr.Code
		if description := strings.TrimSpace(oauthErr.Description); description != "" {
			payload["error_description"] = description
		}
		if authReq != nil && strings.TrimSpace(authReq.State) != "" {
			payload["state"] = authReq.State
		}
		title = "Authorization denied"
		message = "We sent the authorization result back to your app."
	} else {
		parsedRedirect, parseErr := url.Parse(strings.TrimSpace(redirectURL))
		if parseErr != nil {
			return false
		}
		if code := strings.TrimSpace(parsedRedirect.Query().Get("code")); code != "" {
			payload["code"] = code
		}
		if authReq != nil && strings.TrimSpace(authReq.State) != "" {
			payload["state"] = authReq.State
		}
		if payload["code"] == "" {
			return false
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_ = authorizeWebMessageTemplate.Execute(w, map[string]any{
		"Title":            title,
		"Message":          message,
		"PayloadJSON":      mustJSONJS(payload),
		"TargetOriginJSON": mustJSONJS(targetOrigin),
		"ResponseModeJSON": mustJSONJS(strings.TrimSpace(authReq.ResponseMode)),
	})
	return true
}

func (api *OAuthAPI) handleCORS(w http.ResponseWriter, r *http.Request, allowMethods string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && !api.originAllowed(origin) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return true
	}
	if origin != "" {
		api.setCORSHeaders(w, r, origin, allowMethods)
	}

	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", allowMethods)
		w.WriteHeader(http.StatusNoContent)
		return true
	}

	return false
}

func (api *OAuthAPI) originAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || len(api.allowedOrigins) == 0 {
		return true
	}

	_, ok := api.allowedOrigins[origin]
	return ok
}

func (api *OAuthAPI) setCORSHeaders(w http.ResponseWriter, r *http.Request, origin, allowMethods string) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	addVaryHeader(w, "Origin")
	w.Header().Set("Access-Control-Allow-Methods", allowMethods)
	w.Header().Set("Access-Control-Allow-Headers", oauthRequestedHeaders(r))
	if strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers")) != "" {
		addVaryHeader(w, "Access-Control-Request-Headers")
	}
}

func oauthRequestedHeaders(r *http.Request) string {
	if r == nil {
		return oauthCORSDefaultAllowedHeaders
	}
	if requested := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers")); requested != "" {
		return requested
	}
	return oauthCORSDefaultAllowedHeaders
}

func addVaryHeader(w http.ResponseWriter, value string) {
	for _, existing := range w.Header().Values("Vary") {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	w.Header().Add("Vary", value)
}

func (api *OAuthAPI) logOAuthAuthorizeSuccess(r *http.Request, authReq *auth.OAuthAuthorizationRequest, redirectURL string) {
	if api == nil || api.logger == nil {
		return
	}
	fields := oauthRequestFields(r)
	mergeFieldMaps(fields, oauthAuthorizationFields(authReq))
	fields["redirect_destination_host"] = hostFromURL(redirectURL)
	fields["decision"] = "approve"
	api.logger.Info("OAuth authorize redirecting to client callback", logging.WithFields(fields))
}

func (api *OAuthAPI) logOAuthAuthorizeFailure(r *http.Request, authReq *auth.OAuthAuthorizationRequest, err error) {
	if api == nil || api.logger == nil {
		return
	}
	fields := oauthRequestFields(r)
	mergeFieldMaps(fields, oauthAuthorizationFields(authReq))
	mergeFieldMaps(fields, oauthErrorFields(err))
	fields["decision"] = strings.ToLower(strings.TrimSpace(r.PostFormValue("decision")))
	api.logger.Warn("OAuth authorize failed", logging.WithFields(fields))
}

func (api *OAuthAPI) logOAuthCallbackSuccess(r *http.Request, redirectTo string) {
	if api == nil || api.logger == nil {
		return
	}
	fields := oauthRequestFields(r)
	fields["redirect_destination_host"] = hostFromURL(redirectTo)
	fields["has_code"] = strings.TrimSpace(r.URL.Query().Get("code")) != ""
	fields["has_state"] = strings.TrimSpace(r.URL.Query().Get("state")) != ""
	api.logger.Info("OAuth Google callback restored session", logging.WithFields(fields))
}

func (api *OAuthAPI) logOAuthCallbackFailure(r *http.Request, err error) {
	if api == nil || api.logger == nil {
		return
	}
	fields := oauthRequestFields(r)
	fields["has_code"] = strings.TrimSpace(r.URL.Query().Get("code")) != ""
	fields["has_state"] = strings.TrimSpace(r.URL.Query().Get("state")) != ""
	mergeFieldMaps(fields, oauthErrorFields(err))
	api.logger.Warn("OAuth Google callback failed", logging.WithFields(fields))
}

func (api *OAuthAPI) logOAuthTokenSuccess(r *http.Request, form url.Values) {
	if api == nil || api.logger == nil {
		return
	}
	fields := oauthRequestFields(r)
	mergeFieldMaps(fields, oauthTokenRequestFields(form))
	api.logger.Info("OAuth token exchange succeeded", logging.WithFields(fields))
}

func (api *OAuthAPI) logOAuthTokenFailure(r *http.Request, form url.Values, err error) {
	if api == nil || api.logger == nil {
		return
	}
	fields := oauthRequestFields(r)
	mergeFieldMaps(fields, oauthTokenRequestFields(form))
	mergeFieldMaps(fields, oauthErrorFields(err))
	api.logger.Warn("OAuth token exchange failed", logging.WithFields(fields))
}

func oauthRequestFields(r *http.Request) map[string]interface{} {
	fields := map[string]interface{}{}
	if r == nil {
		return fields
	}
	fields["method"] = r.Method
	fields["path"] = r.URL.Path
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		fields["origin"] = origin
	}
	return fields
}

func oauthAuthorizationFields(authReq *auth.OAuthAuthorizationRequest) map[string]interface{} {
	fields := map[string]interface{}{}
	if authReq == nil {
		return fields
	}
	fields["client_id"] = authReq.ClientID
	fields["redirect_uri_host"] = hostFromURL(authReq.RedirectURI)
	fields["has_state"] = strings.TrimSpace(authReq.State) != ""
	fields["resource"] = strings.TrimSpace(authReq.Resource)
	fields["response_mode"] = strings.TrimSpace(authReq.ResponseMode)
	fields["scope"] = strings.TrimSpace(authReq.Scope)
	return fields
}

func oauthTokenRequestFields(form url.Values) map[string]interface{} {
	fields := map[string]interface{}{}
	if form == nil {
		return fields
	}
	fields["grant_type"] = strings.TrimSpace(form.Get("grant_type"))
	fields["client_id"] = strings.TrimSpace(form.Get("client_id"))
	fields["redirect_uri_host"] = hostFromURL(form.Get("redirect_uri"))
	fields["resource"] = strings.TrimSpace(form.Get("resource"))
	fields["has_code"] = strings.TrimSpace(form.Get("code")) != ""
	fields["has_code_verifier"] = strings.TrimSpace(form.Get("code_verifier")) != ""
	fields["has_refresh_token"] = strings.TrimSpace(form.Get("refresh_token")) != ""
	return fields
}

func oauthErrorFields(err error) map[string]interface{} {
	oauthErr := auth.NormalizeOAuthError(err)
	return map[string]interface{}{
		"oauth_error":             oauthErr.Code,
		"oauth_error_description": oauthErr.Description,
		"oauth_status":            oauthErr.StatusCode,
	}
}

func hostFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Host)
}

func mergeFieldMaps(dst map[string]interface{}, src map[string]interface{}) {
	for key, value := range src {
		dst[key] = value
	}
}

func oauthCookieSameSite(secure bool) http.SameSite {
	if secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func (api *OAuthAPI) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/oauth",
		HttpOnly: true,
		Secure:   api.oauthService.SecureCookies(),
		SameSite: oauthCookieSameSite(api.oauthService.SecureCookies()),
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
	})
}

func (api *OAuthAPI) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/oauth",
		HttpOnly: true,
		Secure:   api.oauthService.SecureCookies(),
		SameSite: oauthCookieSameSite(api.oauthService.SecureCookies()),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func writeOAuthJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func usesWebMessageResponseMode(authReq *auth.OAuthAuthorizationRequest) bool {
	if authReq == nil {
		return false
	}
	switch strings.TrimSpace(authReq.ResponseMode) {
	case "web_message", "web_message.opener":
		return true
	default:
		return false
	}
}

func originFromRedirectURI(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func mustJSONJS(value interface{}) template.JS {
	encoded, err := json.Marshal(value)
	if err != nil {
		return template.JS(strconv.Quote(""))
	}
	return template.JS(encoded)
}
