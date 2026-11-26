package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/allocator"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/http/handlers"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

type testSigner struct {
	err error
}

func (s testSigner) Sign(
	sessionID string,
	gameID string,
	issuedAt time.Time,
	expiresAt time.Time,
) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	return "test-session-token", nil
}

type failingAllocator struct {
	err error
}

func (a failingAllocator) Allocate(
	ctx context.Context,
	gameID string,
	sessionID string,
) (models.GameServer, error) {
	return models.GameServer{}, a.err
}

func newTestHandler(
	a matchmaking.Allocator,
	s matchmaking.TokenSigner,
) *handlers.SessionHandler {
	service := matchmaking.NewService(
		a,
		s,
		5*time.Minute,
		10*time.Minute,
	)

	return handlers.NewSessionHandler(service)
}

func TestCreateSessionSuccess(t *testing.T) {
	handler := newTestHandler(
		allocator.NewMockAllocator(true),
		testSigner{},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/sessions",
		strings.NewReader(`{"gameId":"doom-1993"}`),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.CreateSession(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("unexpected Content-Type: %s", got)
	}

	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("unexpected Cache-Control: %s", got)
	}

	var response models.CreateSessionResponse

	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.SessionID == "" {
		t.Error("session ID is empty")
	}

	if response.GameID != "doom-1993" {
		t.Errorf("unexpected game ID: %s", response.GameID)
	}

	if response.SessionToken != "test-session-token" {
		t.Error("unexpected session token")
	}

	if response.Server.Host != "203.0.113.10" {
		t.Errorf("unexpected host: %s", response.Server.Host)
	}

	if response.Server.Signaling.Protocol != "wss" {
		t.Error("unexpected signaling protocol")
	}

	if response.Server.Signaling.Port != 32001 {
		t.Error("unexpected signaling port")
	}

	if response.Server.Signaling.Path != "/ws" {
		t.Error("unexpected signaling path")
	}

	if response.Server.WebRTC.Protocol != "udp" {
		t.Error("unexpected WebRTC protocol")
	}

	if response.Server.WebRTC.Port != 32002 {
		t.Error("unexpected WebRTC port")
	}

	if !response.TokenExpiresAt.After(time.Now()) {
		t.Error("token expiration must be in the future")
	}

	if !response.ExpiresAt.After(response.TokenExpiresAt) {
		t.Error("invalid expiration times")
	}
}

func TestCreateSessionInvalidRequests(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantStatus  int
	}{
		{
			name:        "invalid JSON",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"gameId":`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "unsupported game",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"gameId":"unknown-game"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing game ID",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "empty game ID",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"gameId":""}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "unknown field",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"gameId":"doom-1993","extra":true}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "multiple JSON objects",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"gameId":"doom-1993"} {}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        `{"gameId":"doom-1993"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing content type",
			method:      http.MethodPost,
			contentType: "",
			body:        `{"gameId":"doom-1993"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "body too large",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"gameId":"doom-1993"}` + strings.Repeat(" ", 2048),
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "method not allowed",
			method:      http.MethodGet,
			contentType: "application/json",
			body:        "",
			wantStatus:  http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler(
				failingAllocator{
					err: errors.New("unexpected allocation"),
				},
				testSigner{},
			)

			req := httptest.NewRequest(
				tt.method,
				"/api/v1/sessions",
				strings.NewReader(tt.body),
			)

			req.Header.Set("Content-Type", tt.contentType)

			rec := httptest.NewRecorder()

			handler.CreateSession(rec, req)

			assertProblem(
				t,
				rec,
				tt.wantStatus,
				models.CodeInvalidRequest,
			)

			if tt.wantStatus == http.StatusMethodNotAllowed {
				if got := rec.Header().Get("Allow"); got != http.MethodPost {
					t.Errorf("unexpected Allow header: %s", got)
				}
			}
		})
	}
}

func TestCreateSessionServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		allocator  matchmaking.Allocator
		signer     matchmaking.TokenSigner
		wantStatus int
		wantCode   string
		retryAfter string
	}{
		{
			name:       "no available servers",
			allocator:  allocator.NewMockAllocator(false),
			signer:     testSigner{},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   models.CodeNoServers,
			retryAfter: "5",
		},
		{
			name: "allocation failed",
			allocator: failingAllocator{
				err: fmt.Errorf(
					"%w: API timeout",
					matchmaking.ErrAllocationFailed,
				),
			},
			signer:     testSigner{},
			wantStatus: http.StatusInternalServerError,
			wantCode:   models.CodeAllocationFailed,
		},
		{
			name: "internal error",
			allocator: failingAllocator{
				err: errors.New("unexpected internal error"),
			},
			signer:     testSigner{},
			wantStatus: http.StatusInternalServerError,
			wantCode:   models.CodeInternalError,
		},
		{
			name:      "token signing failed",
			allocator: allocator.NewMockAllocator(true),
			signer: testSigner{
				err: errors.New("signing failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   models.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler(tt.allocator, tt.signer)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/sessions",
				strings.NewReader(`{"gameId":"doom-1993"}`),
			)

			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			handler.CreateSession(rec, req)

			assertProblem(
				t,
				rec,
				tt.wantStatus,
				tt.wantCode,
			)

			if got := rec.Header().Get("Retry-After"); got != tt.retryAfter {
				t.Errorf(
					"expected Retry-After %q, got %q",
					tt.retryAfter,
					got,
				)
			}
		})
	}
}

func assertProblem(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	wantStatus int,
	wantCode string,
) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf(
			"expected HTTP %d, got %d: %s",
			wantStatus,
			rec.Code,
			rec.Body.String(),
		)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("unexpected Content-Type: %s", got)
	}

	var problem models.Problem

	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}

	if problem.Status != wantStatus {
		t.Errorf("unexpected problem status: %d", problem.Status)
	}

	if problem.Code != wantCode {
		t.Errorf("expected code %s, got %s", wantCode, problem.Code)
	}

	if problem.Type != "about:blank" {
		t.Errorf("unexpected problem type: %s", problem.Type)
	}

	if problem.Title != http.StatusText(wantStatus) {
		t.Errorf("unexpected problem title: %s", problem.Title)
	}

	if problem.Detail == "" {
		t.Error("problem detail is empty")
	}
}
