package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/http/handlers"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
)

type testTerminator struct {
	sessionID string
	err       error
}

func (t *testTerminator) TerminateSession(
	_ context.Context,
	sessionID string,
) error {
	t.sessionID = sessionID
	return t.err
}

func TestTerminateSessionRequiresAdminToken(t *testing.T) {
	terminator := &testTerminator{}
	handler := handlers.NewAdminHandler(terminator, "admin-secret")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/sessions/session-123/terminate",
		nil,
	)
	req.SetPathValue("sessionID", "session-123")
	rec := httptest.NewRecorder()

	handler.TerminateSession(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if terminator.sessionID != "" {
		t.Fatal("unauthenticated request reached the terminator")
	}
}

func TestTerminateSessionSuccess(t *testing.T) {
	terminator := &testTerminator{}
	handler := handlers.NewAdminHandler(terminator, "admin-secret")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/sessions/session-123/terminate",
		nil,
	)
	req.SetPathValue("sessionID", "session-123")
	req.Header.Set("Authorization", "Bearer admin-secret")
	rec := httptest.NewRecorder()

	handler.TerminateSession(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if terminator.sessionID != "session-123" {
		t.Fatalf("unexpected session ID: %q", terminator.sessionID)
	}
}

func TestTerminateSessionMapsNotFound(t *testing.T) {
	handler := handlers.NewAdminHandler(
		&testTerminator{err: matchmaking.ErrSessionNotFound},
		"admin-secret",
	)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/sessions/missing/terminate",
		nil,
	)
	req.SetPathValue("sessionID", "missing")
	req.Header.Set("Authorization", "Bearer admin-secret")
	rec := httptest.NewRecorder()

	handler.TerminateSession(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTerminateSessionRejectsNonPost(t *testing.T) {
	handler := handlers.NewAdminHandler(&testTerminator{}, "admin-secret")
	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/admin/sessions/session-123/terminate",
		nil,
	)
	req.SetPathValue("sessionID", "session-123")
	req.Header.Set("Authorization", "Bearer admin-secret")
	rec := httptest.NewRecorder()

	handler.TerminateSession(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTerminateSessionMapsUnexpectedError(t *testing.T) {
	handler := handlers.NewAdminHandler(
		&testTerminator{err: errors.New("kubernetes unavailable")},
		"admin-secret",
	)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/sessions/session-123/terminate",
		nil,
	)
	req.SetPathValue("sessionID", "session-123")
	req.Header.Set("Authorization", "Bearer admin-secret")
	rec := httptest.NewRecorder()

	handler.TerminateSession(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}
