package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/http/middleware"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins string
		origin         string
		method         string
		preflight      bool
		wantStatus     int
		wantOrigin     string
		wantNext       bool
	}{
		{
			name:           "allowed origin",
			allowedOrigins: "http://localhost:5173",
			origin:         "http://localhost:5173",
			method:         http.MethodPost,
			wantStatus:     http.StatusOK,
			wantOrigin:     "http://localhost:5173",
			wantNext:       true,
		},
		{
			name:           "multiple allowed origins",
			allowedOrigins: "http://localhost:5173, http://localhost:8081",
			origin:         "http://localhost:8081",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantOrigin:     "http://localhost:8081",
			wantNext:       true,
		},
		{
			name:           "disallowed origin",
			allowedOrigins: "http://localhost:5173",
			origin:         "https://unknown.example",
			method:         http.MethodPost,
			wantStatus:     http.StatusOK,
			wantNext:       true,
		},
		{
			name:           "allowed preflight",
			allowedOrigins: "http://localhost:5173",
			origin:         "http://localhost:5173",
			method:         http.MethodOptions,
			preflight:      true,
			wantStatus:     http.StatusNoContent,
			wantOrigin:     "http://localhost:5173",
		},
		{
			name:           "disallowed preflight",
			allowedOrigins: "http://localhost:5173",
			origin:         "https://unknown.example",
			method:         http.MethodOptions,
			preflight:      true,
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "request without origin",
			allowedOrigins: "http://localhost:5173",
			method:         http.MethodPost,
			wantStatus:     http.StatusOK,
			wantNext:       true,
		},
		{
			name:       "CORS disabled",
			origin:     "http://localhost:5173",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		{
			name:           "wildcard not allowed",
			allowedOrigins: "*",
			origin:         "https://unknown.example",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantNext:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.CORS(
				tt.allowedOrigins,
				next,
			)

			req := httptest.NewRequest(
				tt.method,
				"/api/v1/sessions",
				nil,
			)

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			if tt.preflight {
				req.Header.Set(
					"Access-Control-Request-Method",
					http.MethodPost,
				)
				req.Header.Set(
					"Access-Control-Request-Headers",
					"content-type",
				)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf(
					"expected status %d, got %d",
					tt.wantStatus,
					rec.Code,
				)
			}

			if got := rec.Header().Get(
				"Access-Control-Allow-Origin",
			); got != tt.wantOrigin {
				t.Errorf(
					"expected origin %q, got %q",
					tt.wantOrigin,
					got,
				)
			}

			if nextCalled != tt.wantNext {
				t.Errorf(
					"expected nextCalled=%v, got %v",
					tt.wantNext,
					nextCalled,
				)
			}

			if tt.wantOrigin != "" {
				if got := rec.Header().Get(
					"Access-Control-Expose-Headers",
				); got != "Retry-After" {
					t.Errorf("unexpected exposed headers: %q", got)
				}
			}

			if tt.wantStatus == http.StatusNoContent {
				if got := rec.Header().Get(
					"Access-Control-Allow-Methods",
				); got != "GET, POST, OPTIONS" {
					t.Errorf("unexpected allowed methods: %q", got)
				}

				if got := rec.Header().Get(
					"Access-Control-Allow-Headers",
				); got != "Content-Type, Accept" {
					t.Errorf("unexpected allowed headers: %q", got)
				}

				if rec.Body.Len() != 0 {
					t.Error("preflight response must be empty")
				}
			}
		})
	}
}
