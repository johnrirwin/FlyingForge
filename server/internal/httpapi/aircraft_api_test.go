package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

func newTestAircraftAPI() *AircraftAPI {
	return &AircraftAPI{
		logger: logging.New(logging.LevelError),
	}
}

func TestHandleAircraft_MethodNotAllowed(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/aircraft", nil)
	w := httptest.NewRecorder()
	api.handleAircraft(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAircraft_CreateInvalidBody(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/aircraft", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	api.handleAircraft(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAircraftItem_MissingID(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/aircraft/", nil)
	w := httptest.NewRecorder()
	api.handleAircraftItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAircraftItem_UnknownSubResource(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/aircraft/abc/unknown", nil)
	w := httptest.NewRecorder()
	api.handleAircraftItem(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleAircraftItem_DetailsMethodNotAllowed(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/aircraft/abc/details", nil)
	w := httptest.NewRecorder()
	api.handleAircraftItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleComponents_MethodNotAllowed(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodPatch, "/api/aircraft/abc/components", nil)
	w := httptest.NewRecorder()
	api.handleComponents(w, req, "abc")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleComponents_RemoveWithoutCategory(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/aircraft/abc/components", nil)
	w := httptest.NewRecorder()
	api.handleComponents(w, req, "abc")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleComponents_SetInvalidBody(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/aircraft/abc/components", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	api.handleComponents(w, req, "abc")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleReceiver_MethodNotAllowed(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodPatch, "/api/aircraft/abc/receiver", nil)
	w := httptest.NewRecorder()
	api.handleReceiver(w, req, "abc")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleReceiver_SetInvalidBody(t *testing.T) {
	api := newTestAircraftAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/aircraft/abc/receiver", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	api.handleReceiver(w, req, "abc")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
