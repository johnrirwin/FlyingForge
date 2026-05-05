package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
)

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
    code { background: #1f2937; border-radius: 6px; padding: 0.15rem 0.35rem; }
    .actions { display: flex; gap: 0.75rem; margin-top: 1.5rem; }
    button { border: 0; border-radius: 999px; padding: 0.85rem 1.2rem; font-size: 1rem; cursor: pointer; }
    .approve { background: #2563eb; color: white; }
    .deny { background: #374151; color: #f3f4f6; }
    .meta { color: #9ca3af; font-size: 0.95rem; }
  </style>
</head>
<body>
  <main>
    <h1>Authorize {{.Prompt.ClientName}}?</h1>
    <p>This connector is requesting access to your FlyingForge account.</p>
    <ul>
      <li><strong>Client ID:</strong> <code>{{.Prompt.ClientID}}</code></li>
      <li><strong>Redirect URI:</strong> <code>{{.Prompt.RedirectURI}}</code></li>
      {{if .Prompt.Resource}}<li><strong>Resource:</strong> <code>{{.Prompt.Resource}}</code></li>{{end}}
      <li><strong>Requested scopes:</strong> {{.Prompt.Scope}}</li>
    </ul>
    <p class="meta">Only approve this request if you trust this connector to read your FlyingForge aircraft and radio data.</p>
    <form method="post" action="/oauth/authorize">
      <input type="hidden" name="response_type" value="{{.Request.ResponseType}}">
      <input type="hidden" name="client_id" value="{{.Request.ClientID}}">
      <input type="hidden" name="redirect_uri" value="{{.Request.RedirectURI}}">
      <input type="hidden" name="scope" value="{{.Request.Scope}}">
      <input type="hidden" name="state" value="{{.Request.State}}">
      <input type="hidden" name="code_challenge" value="{{.Request.CodeChallenge}}">
      <input type="hidden" name="code_challenge_method" value="{{.Request.CodeChallengeMethod}}">
      <input type="hidden" name="resource" value="{{.Request.Resource}}">
      <div class="actions">
        <button class="approve" type="submit" name="decision" value="approve">Approve</button>
        <button class="deny" type="submit" name="decision" value="deny">Deny</button>
      </div>
    </form>
  </main>
</body>
</html>`))

// OAuthAPI exposes a self-hosted authorization server for MCP.
type OAuthAPI struct {
	oauthService *auth.OAuthServerService
	logger       *logging.Logger
}

func NewOAuthAPI(oauthService *auth.OAuthServerService, logger *logging.Logger) *OAuthAPI {
	return &OAuthAPI{oauthService: oauthService, logger: logger}
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
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeOAuthJSON(w, http.StatusOK, api.oauthService.AuthorizationServerMetadata())
}

func (api *OAuthAPI) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeOAuthJSON(w, http.StatusOK, api.oauthService.JWKS())
}

func (api *OAuthAPI) handleRegisterClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
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
		api.clearCookie(w, api.oauthService.SessionCookieName())
		redirectURL := "/oauth/authorize?" + r.PostForm.Encode()
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	switch strings.ToLower(strings.TrimSpace(r.PostForm.Get("decision"))) {
	case "approve":
		redirectURL, authorizeErr := api.oauthService.Authorize(r.Context(), authReq, userID)
		if authorizeErr != nil {
			api.redirectOrWriteAuthorizeError(w, r, authReq, authorizeErr)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
	case "deny":
		api.redirectOrWriteAuthorizeError(w, r, authReq, &auth.OAuthError{
			Code:        "access_denied",
			Description: "user denied the authorization request",
			StatusCode:  http.StatusUnauthorized,
		})
	default:
		api.redirectOrWriteAuthorizeError(w, r, authReq, &auth.OAuthError{
			Code:        "invalid_request",
			Description: "authorization decision is required",
			StatusCode:  http.StatusBadRequest,
		})
	}
}

func (api *OAuthAPI) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if authError := r.URL.Query().Get("error"); authError != "" {
		api.clearCookie(w, api.oauthService.PendingCookieName())
		api.writeOAuthError(w, &auth.OAuthError{Code: "access_denied", Description: "Google authentication was denied", StatusCode: http.StatusUnauthorized})
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	pendingCookie, err := r.Cookie(api.oauthService.PendingCookieName())
	if err != nil || code == "" || state == "" {
		api.writeOAuthError(w, &auth.OAuthError{Code: "access_denied", Description: "missing OAuth login callback state", StatusCode: http.StatusUnauthorized})
		return
	}

	sessionToken, redirectTo, callbackErr := api.oauthService.HandleGoogleCallback(r.Context(), code, state, pendingCookie.Value)
	if callbackErr != nil {
		api.clearCookie(w, api.oauthService.PendingCookieName())
		api.writeOAuthError(w, callbackErr)
		return
	}

	api.clearCookie(w, api.oauthService.PendingCookieName())
	api.setCookie(w, api.oauthService.SessionCookieName(), sessionToken, api.oauthService.SessionCookieTTL())
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (api *OAuthAPI) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		api.writeOAuthError(w, &auth.OAuthError{Code: "invalid_request", Description: "invalid token request body", StatusCode: http.StatusBadRequest})
		return
	}

	response, err := api.oauthService.ExchangeToken(r.Context(), r.PostForm)
	if err != nil {
		api.writeOAuthError(w, err)
		return
	}
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
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	api.writeOAuthError(w, err)
}

func (api *OAuthAPI) renderAuthorizeConsentPage(w http.ResponseWriter, authReq *auth.OAuthAuthorizationRequest, prompt *auth.OAuthAuthorizationPrompt) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	if err := authorizeConsentTemplate.Execute(w, map[string]any{
		"Prompt":  prompt,
		"Request": authReq,
	}); err != nil {
		http.Error(w, "failed to render authorization prompt", http.StatusInternalServerError)
	}
}

func (api *OAuthAPI) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/oauth",
		HttpOnly: true,
		Secure:   api.oauthService.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
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
		SameSite: http.SameSiteLaxMode,
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
