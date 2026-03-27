package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestSocialAPI() *SocialAPI {
	return &SocialAPI{
		logger: logging.New(logging.LevelError),
	}
}

func TestHandleFollow_MissingTargetUserID(t *testing.T) {
	api := newTestSocialAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/social/follow/", nil)
	w := httptest.NewRecorder()
	api.handleFollow(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleFollow_MethodNotAllowed(t *testing.T) {
	api := newTestSocialAPI()
	req := httptest.NewRequest(http.MethodPatch, "/api/social/follow/user-abc", nil)
	w := httptest.NewRecorder()
	api.handleFollow(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleFollow_Options(t *testing.T) {
	api := newTestSocialAPI()
	req := httptest.NewRequest(http.MethodOptions, "/api/social/follow/user-abc", nil)
	w := httptest.NewRecorder()
	api.handleFollow(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
