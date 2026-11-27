package matchmaking

import (
	"context"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

type Allocator interface {
	Allocate(
		ctx context.Context,
		gameID string,
		sessionID string,
	) (models.GameServer, error)
}

type SessionTerminator interface {
	TerminateSession(ctx context.Context, sessionID string) error
}
