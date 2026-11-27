package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/turncredentials"
)

func TestTURNHandlerReturnsNoStoreCredentials(t *testing.T) {
	service, err := turncredentials.New(
		[]string{"turns:10.93.26.213:5349?transport=tcp"},
		"0123456789abcdef0123456789abcdef",
		time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}

	handler := NewTURNHandler(service)
	response := httptest.NewRecorder()
	handler.GetCredentials(response, httptest.NewRequest(http.MethodGet, "/api/v1/ice-credentials", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache policy, got %q", response.Header().Get("Cache-Control"))
	}

	var credentials turncredentials.ICECredentials
	if err := json.Unmarshal(response.Body.Bytes(), &credentials); err != nil {
		t.Fatal(err)
	}
	if len(credentials.ICEServers) != 1 || credentials.ICEServers[0].Username == "" || credentials.ICEServers[0].Credential == "" {
		t.Fatalf("response does not contain TURN credentials: %#v", credentials)
	}
}

func TestTURNHandlerRejectsNonGet(t *testing.T) {
	service, err := turncredentials.New(
		[]string{"turns:10.93.26.213:5349?transport=tcp"},
		"0123456789abcdef0123456789abcdef",
		time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	NewTURNHandler(service).GetCredentials(
		response,
		httptest.NewRequest(http.MethodPost, "/api/v1/ice-credentials", nil),
	)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("got Allow %q, want GET", got)
	}
}
