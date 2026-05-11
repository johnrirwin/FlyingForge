package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestBatteryAPI() *BatteryAPI {
	return &BatteryAPI{
		logger: logging.New(logging.LevelError),
	}
}

func TestHandleBatteries_MethodNotAllowed(t *testing.T) {
	api := newTestBatteryAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/batteries", nil)
	w := httptest.NewRecorder()
	api.handleBatteries(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBatteries_CreateInvalidBody(t *testing.T) {
	api := newTestBatteryAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/batteries", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	api.handleBatteries(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleBatteryItem_MissingID(t *testing.T) {
	api := newTestBatteryAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/batteries/", nil)
	w := httptest.NewRecorder()
	api.handleBatteryItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleBatteryItem_UnknownSubResource(t *testing.T) {
	api := newTestBatteryAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/batteries/abc/unknown", nil)
	w := httptest.NewRecorder()
	api.handleBatteryItem(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleBatteryItem_DetailsMethodNotAllowed(t *testing.T) {
	api := newTestBatteryAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/batteries/abc/details", nil)
	w := httptest.NewRecorder()
	api.handleBatteryItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBatteryItem_BaseMethodNotAllowed(t *testing.T) {
	api := newTestBatteryAPI()
	req := httptest.NewRequest(http.MethodOptions, "/api/batteries/abc", nil)
	w := httptest.NewRecorder()
	api.handleBatteryItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
