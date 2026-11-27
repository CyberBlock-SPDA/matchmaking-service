package config

import (
	"strings"
	"testing"
)

func TestAgonesRequiresStrongAdminToken(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "agones")
	t.Setenv("MATCHMAKING_ADMIN_TOKEN", "too-short")
	t.Setenv("TURN_URLS", "")
	t.Setenv("TURN_SHARED_SECRET", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "MATCHMAKING_ADMIN_TOKEN") {
		t.Fatalf("expected admin token validation error, got %v", err)
	}
}

func TestAgonesAcceptsAdminTokenAtMinimumLength(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "agones")
	t.Setenv("MATCHMAKING_ADMIN_TOKEN", strings.Repeat("x", 32))
	t.Setenv("TURN_URLS", "")
	t.Setenv("TURN_SHARED_SECRET", "")

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}
}

func TestMockModeDoesNotRequireAdminToken(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "mock")
	t.Setenv("MATCHMAKING_ADMIN_TOKEN", "")
	t.Setenv("TURN_URLS", "")
	t.Setenv("TURN_SHARED_SECRET", "")

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}
}

func TestTURNConfigurationRequiresStrongSecret(t *testing.T) {
	t.Setenv("ALLOCATOR_MODE", "mock")
	t.Setenv("TURN_URLS", "turns:10.93.26.213:5349?transport=tcp")
	t.Setenv("TURN_SHARED_SECRET", "short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TURN_SHARED_SECRET") {
		t.Fatalf("expected TURN secret validation error, got %v", err)
	}
}
