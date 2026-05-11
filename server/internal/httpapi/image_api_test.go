package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestImageAPI() *ImageAPI {
	return &ImageAPI{
		logger: logging.New(logging.LevelError),
	}
}

func TestHandleUpload_MethodNotAllowed(t *testing.T) {
	api := newTestImageAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/images/upload", nil)
	w := httptest.NewRecorder()
	api.handleUpload(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleUpload_Options(t *testing.T) {
	api := newTestImageAPI()
	req := httptest.NewRequest(http.MethodOptions, "/api/images/upload", nil)
	w := httptest.NewRecorder()
	api.handleUpload(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetImage_MethodNotAllowed(t *testing.T) {
	api := newTestImageAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/images/some-id", nil)
	w := httptest.NewRecorder()
	api.handleGetImage(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGetImage_Options(t *testing.T) {
	api := newTestImageAPI()
	req := httptest.NewRequest(http.MethodOptions, "/api/images/some-id", nil)
	w := httptest.NewRecorder()
	api.handleGetImage(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetImage_MissingID(t *testing.T) {
	api := newTestImageAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/images/", nil)
	w := httptest.NewRecorder()
	api.handleGetImage(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
