package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/aggregator"
	"github.com/johnrirwin/flyingforge/internal/logging"
)

type mockRateLimiter struct {
	allow bool
}

func (m *mockRateLimiter) Allow(key string) bool { return m.allow }

func newTestServer(allow bool) *Server {
	return &Server{
		agg:            aggregator.New(nil, nil, nil, logging.New(logging.LevelError)),
		logger:         logging.New(logging.LevelError),
		refreshLimiter: &mockRateLimiter{allow: allow},
	}
}

func TestHandleGetItems_MethodNotAllowed(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	w := httptest.NewRecorder()
	s.handleGetItems(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGetItems_DefaultParams(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	w := httptest.NewRecorder()
	s.handleGetItems(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetItems_WithFilterParams(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodGet,
		"/api/items?limit=10&offset=5&sources=a,b&sourceType=video&q=drone&sort=newest&fromDate=2024-01-01&toDate=2024-12-31&tag=fpv",
		nil)
	w := httptest.NewRecorder()
	s.handleGetItems(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGetSources_MethodNotAllowed(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodPost, "/api/sources", nil)
	w := httptest.NewRecorder()
	s.handleGetSources(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGetSources_Success(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	w := httptest.NewRecorder()
	s.handleGetSources(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := body["sources"]; !ok {
		t.Error("response missing 'sources' key")
	}
	if _, ok := body["count"]; !ok {
		t.Error("response missing 'count' key")
	}
}

func TestHandleRefresh_MethodNotAllowed(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	w := httptest.NewRecorder()
	s.handleRefresh(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleRefresh_RateLimited(t *testing.T) {
	s := newTestServer(false)
	req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	w := httptest.NewRecorder()
	s.handleRefresh(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestHandleRefresh_Success(t *testing.T) {
	s := newTestServer(true)
	req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	w := httptest.NewRecorder()
	s.handleRefresh(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleHealth(t *testing.T) {
	s := &Server{logger: logging.New(logging.LevelError)}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("status = %q, want %q", body["status"], "healthy")
	}
}
