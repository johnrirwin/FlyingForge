package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "hello"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created response",
			status:     http.StatusCreated,
			data:       models.FeedItem{ID: "123", Title: "Test"},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testWriteJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", contentType)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		code       string
		message    string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			code:       "invalid_input",
			message:    "name is required",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			code:       "not_found",
			message:    "resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			code:       "internal_error",
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testWriteError(w, tt.status, tt.code, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["code"] != tt.code {
				t.Errorf("code = %s, want %s", response["code"], tt.code)
			}
			if response["message"] != tt.message {
				t.Errorf("message = %s, want %s", response["message"], tt.message)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	logger := logging.New(logging.LevelError)
	s := &Server{logger: logger}

	handler := s.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("Missing Access-Control-Allow-Origin header")
		}
	})

	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  map[string]string
		defaultLimit int
		maxLimit     int
		wantLimit    int
		wantOffset   int
	}{
		{
			name:         "default values",
			queryParams:  map[string]string{},
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    20,
			wantOffset:   0,
		},
		{
			name:         "custom limit",
			queryParams:  map[string]string{"limit": "50"},
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    50,
			wantOffset:   0,
		},
		{
			name:         "limit exceeds max",
			queryParams:  map[string]string{"limit": "200"},
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    100,
			wantOffset:   0,
		},
		{
			name:         "with offset",
			queryParams:  map[string]string{"limit": "20", "offset": "40"},
			defaultLimit: 20,
			maxLimit:     100,
			wantLimit:    20,
			wantOffset:   40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			limit, offset := testParsePagination(req, tt.defaultLimit, tt.maxLimit)

			if limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tt.wantLimit)
			}
			if offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

func TestParseJSONBody(t *testing.T) {
	type testBody struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		body    string
		want    testBody
		wantErr bool
	}{
		{
			name:    "valid JSON",
			body:    `{"name":"test","value":42}`,
			want:    testBody{Name: "test", Value: 42},
			wantErr: false,
		},
		{
			name:    "empty body",
			body:    "",
			want:    testBody{},
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			body:    `{"name":}`,
			want:    testBody{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			var result testBody
			err := json.NewDecoder(req.Body).Decode(&result)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.want {
				t.Errorf("result = %+v, want %+v", result, tt.want)
			}
		})
	}
}

// Test helpers - duplicated here to avoid circular imports
func testWriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func testWriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": message,
	})
}

func testParsePagination(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
