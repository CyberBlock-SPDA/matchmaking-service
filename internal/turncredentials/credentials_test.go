package turncredentials

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestCredentialsUseCoturnRESTAuthentication(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	service, err := New(
		[]string{"turns:10.93.26.213:5349?transport=tcp"},
		secret,
		10*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time {
		return time.Unix(1_800_000_000, 0)
	}

	got, err := service.Credentials()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ICEServers) != 1 {
		t.Fatalf("got %d ICE servers, want 1", len(got.ICEServers))
	}

	server := got.ICEServers[0]
	if len(server.URLs) != 1 || server.URLs[0] != "turns:10.93.26.213:5349?transport=tcp" {
		t.Fatalf("unexpected TURN URLs: %#v", server.URLs)
	}
	if !strings.HasPrefix(server.Username, "1800000600:") {
		t.Fatalf("unexpected credential expiry in username %q", server.Username)
	}

	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(server.Username))
	if want := base64.StdEncoding.EncodeToString(mac.Sum(nil)); server.Credential != want {
		t.Fatalf("credential does not match coturn REST authentication")
	}
}

func TestNewRejectsWeakConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		urls   []string
		secret string
		ttl    time.Duration
	}{
		{name: "missing URL", secret: strings.Repeat("x", 32), ttl: time.Minute},
		{name: "weak secret", urls: []string{"turn:localhost:3478"}, secret: "short", ttl: time.Minute},
		{name: "invalid scheme", urls: []string{"stun:localhost:3478"}, secret: strings.Repeat("x", 32), ttl: time.Minute},
		{name: "invalid TTL", urls: []string{"turn:localhost:3478"}, secret: strings.Repeat("x", 32)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.urls, test.secret, test.ttl); err == nil {
				t.Fatal("expected invalid TURN configuration to fail")
			}
		})
	}
}
