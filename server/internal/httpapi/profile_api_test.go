package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestProfileAPI() *ProfileAPI {
	return &ProfileAPI{
		logger: logging.New(logging.LevelError),
	}
}

func TestHandleProfile_MethodNotAllowed(t *testing.T) {
	api := newTestProfileAPI()
	req := httptest.NewRequest(http.MethodPatch, "/api/me/profile", nil)
	w := httptest.NewRecorder()
	api.handleProfile(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleProfile_POST_MethodNotAllowed(t *testing.T) {
	api := newTestProfileAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/me/profile", nil)
	w := httptest.NewRecorder()
	api.handleProfile(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
