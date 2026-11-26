package matchmaking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

type TokenSigner interface {
	Sign(
		sessionID string,
		gameID string,
		issuedAt time.Time,
		expiresAt time.Time,
	) (string, error)
}

type Service struct {
	allocator  Allocator
	signer     TokenSigner
	tokenTTL   time.Duration
	sessionTTL time.Duration
}

func NewService(
	allocator Allocator,
	signer TokenSigner,
	tokenTTL time.Duration,
	sessionTTL time.Duration,
) *Service {
	return &Service{
		allocator:  allocator,
		signer:     signer,
		tokenTTL:   tokenTTL,
		sessionTTL: sessionTTL,
	}
}

func (s *Service) CreateSession(
	ctx context.Context,
	gameID string,
) (models.CreateSessionResponse, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return models.CreateSessionResponse{}, err
	}

	server, err := s.allocator.Allocate(ctx, gameID)
	if err != nil {
		return models.CreateSessionResponse{}, err
	}

	now := time.Now().UTC().Truncate(time.Second)

	tokenExpiresAt := now.Add(s.tokenTTL)
	expiresAt := now.Add(s.sessionTTL)

	token, err := s.signer.Sign(
		sessionID,
		gameID,
		now,
		tokenExpiresAt,
	)
	if err != nil {
		return models.CreateSessionResponse{}, err
	}

	return models.CreateSessionResponse{
		SessionID:      sessionID,
		GameID:         gameID,
		Server:         server,
		SessionToken:   token,
		TokenExpiresAt: tokenExpiresAt,
		ExpiresAt:      expiresAt,
	}, nil
}

func generateSessionID() (string, error) {
	var randomBytes [16]byte

	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", err
	}

	return "session-" + hex.EncodeToString(randomBytes[:]), nil
}
