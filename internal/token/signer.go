package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
)

type Signer struct {
	privateKey ed25519.PrivateKey
}

func NewSigner(keyPath string) (*Signer, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	key, err := jwt.ParseEdPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected Ed25519 private key")
	}

	return &Signer{
		privateKey: privateKey,
	}, nil
}

func (s *Signer) Sign(
	sessionID string,
	gameID string,
	issuedAt time.Time,
	expiresAt time.Time,
) (string, error) {
	if sessionID == "" || gameID == "" {
		return "", fmt.Errorf("session ID and game ID are required")
	}

	if !expiresAt.After(issuedAt) {
		return "", fmt.Errorf("invalid token expiration time")
	}

	var randomBytes [16]byte

	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", fmt.Errorf("generate token ID: %w", err)
	}

	claims := SessionClaims{
		GameID: gameID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
			Subject:   sessionID,
			ID:        hex.EncodeToString(randomBytes[:]),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodEdDSA,
		claims,
	)

	return token.SignedString(s.privateKey)
}

var _ matchmaking.TokenSigner = (*Signer)(nil)
