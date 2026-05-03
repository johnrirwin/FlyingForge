package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	appauth "github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
)

type authCodeError interface {
	error
	ErrorCode() string
}

type HTTPAuthProvider interface {
	Enabled() bool
	AuthenticateBearerToken(ctx context.Context, token string) (string, error)
	Challenge(errorCode, description string) string
	ProtectedResourceMetadata() *appauth.MCPProtectedResourceMetadata
}

type HTTPHandler struct {
	protocol       *Protocol
	authProvider   HTTPAuthProvider
	allowedOrigins map[string]struct{}
	logger         *logging.Logger
}

func NewHTTPHandler(protocol *Protocol, authProvider HTTPAuthProvider, allowedOrigins []string, logger *logging.Logger) *HTTPHandler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			originSet[trimmed] = struct{}{}
		}
	}

	return &HTTPHandler{
		protocol:       protocol,
		authProvider:   authProvider,
		allowedOrigins: originSet,
		logger:         logger,
	}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !h.originAllowed(origin) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	h.setCORSHeaders(w, origin)

	switch r.Method {
	case http.MethodOptions:
		w.Header().Set("Allow", "POST, GET, DELETE, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodGet:
		w.Header().Set("Allow", "POST, GET, DELETE, OPTIONS")
		http.Error(w, "streaming GET not supported", http.StatusMethodNotAllowed)
	case http.MethodDelete:
		w.Header().Set("Allow", "POST, GET, DELETE, OPTIONS")
		http.Error(w, "session termination is not supported", http.StatusMethodNotAllowed)
	default:
		w.Header().Set("Allow", "POST, GET, DELETE, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPHandler) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if h.authProvider == nil || !h.authProvider.Enabled() {
		http.NotFound(w, r)
		return
	}

	origin := r.Header.Get("Origin")
	if !h.originAllowed(origin) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	h.setCORSHeaders(w, origin)

	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, h.authProvider.ProtectedResourceMetadata())
}

func (h *HTTPHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	body, err := readAllJSON(r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	authState := RequestAuth{}
	if h.requestNeedsAuthentication(body) {
		authState = h.authenticateRequest(ctx, r.Header.Get("Authorization"))
	}
	ctx = WithRequestAuth(ctx, authState)

	response := h.protocol.HandleMessage(ctx, body)
	if response == nil {
		if authState.Challenge != "" {
			w.Header().Set("WWW-Authenticate", authState.Challenge)
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if authState.Challenge != "" {
		w.Header().Set("WWW-Authenticate", authState.Challenge)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *HTTPHandler) requestNeedsAuthentication(body []byte) bool {
	if h.authProvider == nil || !h.authProvider.Enabled() {
		return false
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	if req.Method != "tools/call" {
		return false
	}

	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return false
	}

	if h.protocol == nil || h.protocol.handler == nil {
		return false
	}

	return h.protocol.handler.IsPrivateTool(params.Name)
}

func (h *HTTPHandler) authenticateRequest(ctx context.Context, authorizationHeader string) RequestAuth {
	if h.authProvider == nil || !h.authProvider.Enabled() {
		return RequestAuth{}
	}

	token := bearerToken(authorizationHeader)
	if token == "" {
		return RequestAuth{
			Challenge:          h.authProvider.Challenge("invalid_token", "Authentication required: no access token provided."),
			ChallengeMessage:   "Authentication required: no access token provided.",
			ChallengeErrorCode: "invalid_token",
		}
	}

	userID, err := h.authProvider.AuthenticateBearerToken(ctx, token)
	if err == nil {
		return RequestAuth{UserID: userID}
	}

	code := "invalid_token"
	var typedErr authCodeError
	if errors.As(err, &typedErr) && typedErr.ErrorCode() != "" {
		code = typedErr.ErrorCode()
	}

	return RequestAuth{
		Challenge:          h.authProvider.Challenge(code, err.Error()),
		ChallengeMessage:   err.Error(),
		ChallengeErrorCode: code,
	}
}

func (h *HTTPHandler) originAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || len(h.allowedOrigins) == 0 {
		return true
	}

	_, ok := h.allowedOrigins[origin]
	return ok
}

func (h *HTTPHandler) setCORSHeaders(w http.ResponseWriter, origin string) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Last-Event-ID, MCP-Session-Id")
}

func readAllJSON(r *http.Request) ([]byte, error) {
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(io.LimitReader(r.Body, 1<<20)); err != nil {
		return nil, err
	}
	body := bytes.TrimSpace(buffer.Bytes())
	if len(body) == 0 {
		return nil, errors.New("empty body")
	}
	return body, nil
}

func bearerToken(authorizationHeader string) string {
	parts := strings.SplitN(strings.TrimSpace(authorizationHeader), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
