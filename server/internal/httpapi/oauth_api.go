package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
)

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
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authReq, err := api.oauthService.ParseAuthorizationRequest(r.URL.Query())
	if err != nil {
		api.writeOAuthError(w, err)
		return
	}

	if sessionCookie, err := r.Cookie(api.oauthService.SessionCookieName()); err == nil {
		if userID, sessionErr := api.oauthService.ValidateSessionToken(sessionCookie.Value); sessionErr == nil {
			redirectURL, authorizeErr := api.oauthService.Authorize(r.Context(), authReq, userID)
			if authorizeErr != nil {
				api.writeOAuthError(w, authorizeErr)
				return
			}
			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}
		api.clearCookie(w, api.oauthService.SessionCookieName())
	}

	googleURL, pendingToken, err := api.oauthService.BuildGoogleAuthorizationURL(r.URL.RequestURI())
	if err != nil {
		api.writeOAuthError(w, err)
		return
	}
	api.setCookie(w, api.oauthService.PendingCookieName(), pendingToken, api.oauthService.PendingCookieTTL())
	http.Redirect(w, r, googleURL, http.StatusFound)
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
	oauthErr := normalizeOAuthError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(oauthErr.StatusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             oauthErr.Code,
		"error_description": oauthErr.Description,
	})
}

func normalizeOAuthError(err error) *auth.OAuthError {
	if err == nil {
		return &auth.OAuthError{Code: "server_error", Description: "unknown OAuth error", StatusCode: http.StatusInternalServerError}
	}
	typed, ok := err.(*auth.OAuthError)
	if ok {
		if typed.StatusCode == 0 {
			typed.StatusCode = http.StatusBadRequest
		}
		return typed
	}
	return &auth.OAuthError{Code: "server_error", Description: err.Error(), StatusCode: http.StatusInternalServerError}
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
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
