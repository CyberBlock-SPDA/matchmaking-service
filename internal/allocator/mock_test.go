package allocator

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
)

func TestMockAllocatorSuccess(t *testing.T) {
	a := NewMockAllocator(true)

	server, err := a.Allocate(
		context.Background(),
		"doom-1993",
		"session-test-123",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if server.Host != "203.0.113.10" {
		t.Errorf("unexpected host: %s", server.Host)
	}

	if server.Signaling.Port != 32001 {
		t.Errorf("unexpected signaling port: %d",
			server.Signaling.Port)
	}

	if server.WebRTC.Port != 32002 {
		t.Errorf("unexpected WebRTC port: %d",
			server.WebRTC.Port)
	}
}

func TestMockAllocatorNoServers(t *testing.T) {
	a := NewMockAllocator(false)

	_, err := a.Allocate(
		context.Background(),
		"doom-1993",
		"session-test-123",
	)

	if !errors.Is(err, matchmaking.ErrNoAvailableServers) {
		t.Fatalf("expected ErrNoAvailableServers, got: %v", err)
	}
}

func TestMockAllocatorCanceled(t *testing.T) {
	a := NewMockAllocator(true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.Allocate(ctx, "doom-1993", "session-test-123")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}
