package matchmaking

import (
	"context"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

type Allocator interface {
	Allocate(ctx context.Context, gameID string) (models.GameServer, error)
}
