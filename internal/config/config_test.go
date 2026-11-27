package config

import (
	"strings"
	"testing"
)

func TestAgonesRequiresStrongAdminToken(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "agones")
	t.Setenv("MATCHMAKING_ADMIN_TOKEN", "too-short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "MATCHMAKING_ADMIN_TOKEN") {
		t.Fatalf("expected admin token validation error, got %v", err)
	}
}

func TestAgonesAcceptsAdminTokenAtMinimumLength(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "agones")
	t.Setenv("MATCHMAKING_ADMIN_TOKEN", strings.Repeat("x", 32))

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}
}

func TestMockModeDoesNotRequireAdminToken(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "mock")
	t.Setenv("MATCHMAKING_ADMIN_TOKEN", "")

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}
}
