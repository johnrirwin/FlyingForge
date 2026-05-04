package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	appauth "github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

type stubEquipmentReader struct{}

func (stubEquipmentReader) Search(_ context.Context, params models.EquipmentSearchParams) (*models.EquipmentSearchResponse, error) {
	return &models.EquipmentSearchResponse{
		Items: []models.EquipmentItem{
			{
				ID:           "eq-1",
				Name:         "ExpressLRS Receiver",
				Category:     models.CategoryReceivers,
				Manufacturer: "Acme",
			},
		},
		TotalCount: 1,
		Page:       1,
		PageSize:   max(params.Limit, 1),
		Query:      params.Query,
	}, nil
}

func (stubEquipmentReader) GetByCategory(_ context.Context, category models.EquipmentCategory, limit, _ int) (*models.EquipmentSearchResponse, error) {
	return &models.EquipmentSearchResponse{
		Items: []models.EquipmentItem{
			{
				ID:       "eq-1",
				Name:     "Frame",
				Category: category,
			},
		},
		TotalCount: 1,
		Page:       1,
		PageSize:   max(limit, 1),
	}, nil
}

func (stubEquipmentReader) GetSellers() []models.SellerInfo {
	return []models.SellerInfo{{ID: "gear-catalog", Name: "Gear Catalog", Enabled: true}}
}

type fakeHTTPAuthError struct {
	code string
	msg  string
}

func (e *fakeHTTPAuthError) Error() string     { return e.msg }
func (e *fakeHTTPAuthError) ErrorCode() string { return e.code }

type fakeHTTPAuthProvider struct {
	userID string
	err    error
}

func (p *fakeHTTPAuthProvider) Enabled() bool { return true }

func (p *fakeHTTPAuthProvider) AuthenticateBearerToken(_ context.Context, token string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	if token == "" {
		return "", &fakeHTTPAuthError{code: "invalid_token", msg: "missing token"}
	}
	return p.userID, nil
}

func (p *fakeHTTPAuthProvider) Challenge(errorCode, description string) string {
	return fmt.Sprintf(`Bearer realm="flyingforge", error="%s", error_description="%s"`, errorCode, description)
}

func (p *fakeHTTPAuthProvider) ProtectedResourceMetadata() *appauth.MCPProtectedResourceMetadata {
	return &appauth.MCPProtectedResourceMetadata{
		Resource:             "https://example.com/mcp",
		AuthorizationServers: []string{"https://issuer.example.com"},
		ScopesSupported:      []string{"flyingforge.read"},
	}
}

func newTestHTTPHandler(authProvider HTTPAuthProvider) *HTTPHandler {
	handler := NewHandler(
		nil,
		stubEquipmentReader{},
		nil,
		nil,
		nil,
		[]string{"flyingforge.read"},
		testutil.NullLogger(),
	)

	return NewHTTPHandler(NewProtocol(handler, testutil.NullLogger()), authProvider, nil, testutil.NullLogger())
}

func TestHTTPHandlerInitializeDoesNotChallengeWithoutAuthToken(t *testing.T) {
	handler := newTestHTTPHandler(&fakeHTTPAuthProvider{userID: "user-1"})

	requestBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","clientInfo":{"name":"tester","version":"1.0.0"}}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}
	if got := responseRecorder.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no auth challenge for initialize, got %q", got)
	}

	var response struct {
		Result InitializeResult `json:"result"`
		Error  *RPCError        `json:"error"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode initialize response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("expected no RPC error, got %+v", response.Error)
	}
	if response.Result.ProtocolVersion != protocolVersion {
		t.Fatalf("expected protocol version %q, got %q", protocolVersion, response.Result.ProtocolVersion)
	}
}

func TestHTTPHandlerToolsListIncludesPublicAndPrivateTools(t *testing.T) {
	handler := newTestHTTPHandler(&fakeHTTPAuthProvider{userID: "user-1"})

	requestBody := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}
	if got := responseRecorder.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no auth challenge for tools/list, got %q", got)
	}

	var response struct {
		Result ToolsListResult `json:"result"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	foundPublic := false
	foundPrivate := false
	for _, tool := range response.Result.Tools {
		switch tool.Name {
		case "search_equipment":
			foundPublic = true
		case "list_my_aircraft":
			foundPrivate = true
			if len(tool.SecuritySchemes) == 0 || tool.SecuritySchemes[0].Type != "oauth2" {
				t.Fatalf("expected oauth2 security scheme for private tool, got %+v", tool.SecuritySchemes)
			}
		}
	}

	if !foundPublic {
		t.Fatal("expected search_equipment to be listed")
	}
	if !foundPrivate {
		t.Fatal("expected list_my_aircraft to be listed")
	}
}

func TestHTTPHandlerAllowsPublicToolCallsWithoutToken(t *testing.T) {
	handler := newTestHTTPHandler(&fakeHTTPAuthProvider{userID: "user-1"})

	requestBody := []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_equipment","arguments":{"query":"receiver"}}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}
	if got := responseRecorder.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no auth challenge for public tool, got %q", got)
	}

	var response struct {
		Result CallToolResult `json:"result"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode tools/call response: %v", err)
	}
	if response.Result.IsError {
		t.Fatalf("expected successful public tool result, got error result %+v", response.Result)
	}
	if len(response.Result.Content) == 0 {
		t.Fatal("expected text content for public tool result")
	}
}

func TestHTTPHandlerChallengesPrivateToolWithoutToken(t *testing.T) {
	handler := newTestHTTPHandler(&fakeHTTPAuthProvider{userID: "user-1"})

	requestBody := []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_my_aircraft","arguments":{}}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}
	if got := responseRecorder.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatal("expected auth challenge header for private tool call without token")
	}

	var response struct {
		Result CallToolResult `json:"result"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode tools/call response: %v", err)
	}
	if !response.Result.IsError {
		t.Fatalf("expected private tool result to be marked as error, got %+v", response.Result)
	}
	if response.Result.Meta == nil {
		t.Fatal("expected auth challenge metadata on private tool error")
	}
	if _, ok := response.Result.Meta["mcp/www_authenticate"]; !ok {
		t.Fatalf("expected mcp/www_authenticate metadata, got %+v", response.Result.Meta)
	}
}

func TestHTTPHandlerProtectedResourceMetadata(t *testing.T) {
	handler := newTestHTTPHandler(&fakeHTTPAuthProvider{userID: "user-1"})

	request := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	responseRecorder := httptest.NewRecorder()

	handler.HandleProtectedResourceMetadata(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", responseRecorder.Code)
	}

	var metadata appauth.MCPProtectedResourceMetadata
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("failed to decode metadata response: %v", err)
	}
	if metadata.Resource != "https://example.com/mcp" {
		t.Fatalf("unexpected resource metadata: %+v", metadata)
	}
}

func TestHTTPHandlerOptionsPreflight(t *testing.T) {
	handler := newTestHTTPHandler(&fakeHTTPAuthProvider{userID: "user-1"})

	request := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	request.Header.Set("Origin", "https://chatgpt.com")
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected HTTP 204, got %d", responseRecorder.Code)
	}
	if got := responseRecorder.Header().Get("Access-Control-Allow-Origin"); got != "https://chatgpt.com" {
		t.Fatalf("expected CORS origin to echo request origin, got %q", got)
	}
}
