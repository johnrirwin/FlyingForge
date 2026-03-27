package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestGearCatalogAPI() *GearCatalogAPI {
	return &GearCatalogAPI{
		logger: logging.New(logging.LevelError),
	}
}

func TestHandleGearCatalogSearch_MethodNotAllowed(t *testing.T) {
	api := newTestGearCatalogAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/gear-catalog/search", nil)
	w := httptest.NewRecorder()
	api.handleSearch(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
