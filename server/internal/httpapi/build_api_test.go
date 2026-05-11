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

func newTestBuildAPI(allow bool) *BuildAPI {
	return &BuildAPI{
		tempRateLimiter: &mockRateLimiter{allow: allow},
		logger:          logging.New(logging.LevelError),
	}
}

func reqWithUser(method, target string, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	return req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, "user-123"))
}

func TestHandlePublicBuilds_MethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodPost, "/api/public/builds", nil)
	w := httptest.NewRecorder()
	api.handlePublicBuilds(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePublicBuildItem_MissingID(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodGet, "/api/public/builds/", nil)
	w := httptest.NewRecorder()
	api.handlePublicBuildItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePublicBuildItem_UnknownAction(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodGet, "/api/public/builds/123/unknown", nil)
	w := httptest.NewRecorder()
	api.handlePublicBuildItem(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlePublicBuildItem_ImageMethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodPost, "/api/public/builds/123/image", nil)
	w := httptest.NewRecorder()
	api.handlePublicBuildItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleTempCollection_MethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodGet, "/api/builds/temp", nil)
	w := httptest.NewRecorder()
	api.handleTempCollection(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleTempCollection_RateLimited(t *testing.T) {
	api := newTestBuildAPI(false)
	req := httptest.NewRequest(http.MethodPost, "/api/builds/temp", nil)
	w := httptest.NewRecorder()
	api.handleTempCollection(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestHandleTempItem_MissingToken(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodGet, "/api/builds/temp/", nil)
	w := httptest.NewRecorder()
	api.handleTempItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTempItem_UnknownAction(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodPost, "/api/builds/temp/abc123/unknown", nil)
	w := httptest.NewRecorder()
	api.handleTempItem(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleTempItem_ShareMethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := httptest.NewRequest(http.MethodGet, "/api/builds/temp/abc123/share", nil)
	w := httptest.NewRecorder()
	api.handleTempItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBuildCollection_MethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodDelete, "/api/builds", "")
	w := httptest.NewRecorder()
	api.handleBuildCollection(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBuildFromAircraft_MethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodGet, "/api/builds/from-aircraft/abc", "")
	w := httptest.NewRecorder()
	api.handleBuildFromAircraft(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBuildFromAircraft_MissingAircraftID(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodPost, "/api/builds/from-aircraft/", "")
	w := httptest.NewRecorder()
	api.handleBuildFromAircraft(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleBuildItem_MissingID(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodGet, "/api/builds/", "")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleBuildItem_UnknownAction(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodGet, "/api/builds/abc/unknown", "")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleBuildItem_ReactionMethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodPatch, "/api/builds/abc/reaction", "")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBuildItem_ReactionInvalidBody(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodPost, "/api/builds/abc/reaction", "{bad json")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleBuildItem_PublishMethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodGet, "/api/builds/abc/publish", "")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBuildItem_UnpublishMethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodGet, "/api/builds/abc/unpublish", "")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBuildItem_ItemMethodNotAllowed(t *testing.T) {
	api := newTestBuildAPI(true)
	req := reqWithUser(http.MethodPatch, "/api/builds/abc", "")
	w := httptest.NewRecorder()
	api.handleBuildItem(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
