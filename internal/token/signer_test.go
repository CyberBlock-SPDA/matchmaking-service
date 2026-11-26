package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSignerSuccess(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	signer := &Signer{
		privateKey: privateKey,
	}

	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(5 * time.Minute)

	signedToken, err := signer.Sign(
		"session-123",
		"doom-1993",
		now,
		expiresAt,
	)

	if err != nil {
		t.Fatal(err)
	}

	claims := &SessionClaims{}

	parsed, err := jwt.ParseWithClaims(
		signedToken,
		claims,
		func(token *jwt.Token) (any, error) {
			return publicKey, nil
		},
		jwt.WithValidMethods([]string{"EdDSA"}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		t.Fatalf("token verification failed: %v", err)
	}

	if !parsed.Valid {
		t.Fatal("token is invalid")
	}

	if claims.Subject != "session-123" {
		t.Error("unexpected session ID")
	}

	if claims.GameID != "doom-1993" {
		t.Error("unexpected game ID")
	}

	if claims.ID == "" {
		t.Error("token ID is empty")
	}

	if claims.ExpiresAt == nil ||
		!claims.ExpiresAt.Time.Equal(expiresAt) {
		t.Error("unexpected expiration time")
	}
}

func TestSignerWrongKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	wrongPublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	signer := &Signer{
		privateKey: privateKey,
	}

	now := time.Now()

	signedToken, err := signer.Sign(
		"session-123",
		"doom-1993",
		now,
		now.Add(5*time.Minute),
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = jwt.Parse(
		signedToken,
		func(token *jwt.Token) (any, error) {
			return wrongPublicKey, nil
		},
		jwt.WithValidMethods([]string{"EdDSA"}),
	)

	if err == nil {
		t.Fatal("token must not be valid with a wrong key")
	}
}

func TestSignerExpiredToken(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	signer := &Signer{
		privateKey: privateKey,
	}

	now := time.Now()

	signedToken, err := signer.Sign(
		"session-123",
		"doom-1993",
		now.Add(-10*time.Minute),
		now.Add(-5*time.Minute),
	)

	if err != nil {
		t.Fatal(err)
	}

	_, err = jwt.Parse(
		signedToken,
		func(token *jwt.Token) (any, error) {
			return publicKey, nil
		},
		jwt.WithValidMethods([]string{"EdDSA"}),
		jwt.WithExpirationRequired(),
	)

	if err == nil {
		t.Fatal("expired token must be rejected")
	}
}
