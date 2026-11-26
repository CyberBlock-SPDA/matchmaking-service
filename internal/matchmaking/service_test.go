package matchmaking_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/allocator"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
)

type mockSigner struct{}

func (m mockSigner) Sign(
	sessionID string,
	gameID string,
	issuedAt time.Time,
	expiresAt time.Time,
) (string, error) {
	return "test-session-token", nil
}

func TestCreateSessionSuccess(t *testing.T) {
	service := matchmaking.NewService(
		allocator.NewMockAllocator(true),
		mockSigner{},
		5*time.Minute,
		10*time.Minute,
	)

	response, err := service.CreateSession(
		context.Background(),
		"doom-1993",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.SessionID == "" {
		t.Error("session ID is empty")
	}

	if response.GameID != "doom-1993" {
		t.Errorf("unexpected game ID: %s", response.GameID)
	}

	if response.Server.Host != "203.0.113.10" {
		t.Errorf("unexpected host: %s", response.Server.Host)
	}

	if response.SessionToken != "test-session-token" {
		t.Error("unexpected session token")
	}

	if !response.ExpiresAt.After(response.TokenExpiresAt) {
		t.Error("invalid expiration times")
	}
}

func TestCreateSessionNoServers(t *testing.T) {
	service := matchmaking.NewService(
		allocator.NewMockAllocator(false),
		mockSigner{},
		5*time.Minute,
		10*time.Minute,
	)

	_, err := service.CreateSession(
		context.Background(),
		"doom-1993",
	)

	if !errors.Is(err, matchmaking.ErrNoAvailableServers) {
		t.Fatalf("expected no available servers, got: %v", err)
	}
}

func TestCreateSessionUniqueIDs(t *testing.T) {
	service := matchmaking.NewService(
		allocator.NewMockAllocator(true),
		mockSigner{},
		5*time.Minute,
		10*time.Minute,
	)

	first, err := service.CreateSession(
		context.Background(),
		"doom-1993",
	)
	if err != nil {
		t.Fatal(err)
	}

	second, err := service.CreateSession(
		context.Background(),
		"doom-1993",
	)
	if err != nil {
		t.Fatal(err)
	}

	if first.SessionID == second.SessionID {
		t.Error("session IDs must be unique")
	}
}
