package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestAuthAPI() *AuthAPI {
	return &AuthAPI{
		logger:      logging.New(logging.LevelError),
		frontendURL: "http://localhost:3000",
	}
}

func TestHandleGoogleLogin_MethodNotAllowed(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google", nil)
	w := httptest.NewRecorder()
	api.handleGoogleLogin(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGoogleLogin_InvalidBody(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google", strings.NewReader("{invalid json"))
	w := httptest.NewRecorder()
	api.handleGoogleLogin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAuthRefresh_MethodNotAllowed(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/refresh", nil)
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAuthRefresh_InvalidBody(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAuthRefresh_EmptyToken(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refreshToken":""}`))
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleLogout_MethodNotAllowed(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	api.handleLogout(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLogout_NoUserInContext(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	api.handleLogout(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleGetMe_MethodNotAllowed(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	api.handleGetMe(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGetMe_NoUserInContext(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	api.handleGetMe(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleGoogleCallback_MethodNotAllowed(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google/callback", nil)
	w := httptest.NewRecorder()
	api.handleGoogleCallback(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGoogleCallback_ErrorParam(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?error=access_denied", nil)
	w := httptest.NewRecorder()
	api.handleGoogleCallback(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestHandleGoogleCallback_NoCode(t *testing.T) {
	api := newTestAuthAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback", nil)
	w := httptest.NewRecorder()
	api.handleGoogleCallback(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

// Verify that auth.UserIDKey is usable in tests as expected by other test files.
func TestAuthUserIDKeyIsUsable(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "user-123")
	if got := auth.GetUserID(ctx); got != "user-123" {
		t.Errorf("GetUserID = %q, want %q", got, "user-123")
	}
}
