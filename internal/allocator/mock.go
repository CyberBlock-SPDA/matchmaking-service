package allocator

import (
	"context"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

type MockAllocator struct {
	available bool
}

func NewMockAllocator(available bool) *MockAllocator {
	return &MockAllocator{
		available: available,
	}
}

func (a *MockAllocator) Allocate(
	ctx context.Context,
	gameID string,
	sessionID string,
) (models.GameServer, error) {
	if err := ctx.Err(); err != nil {
		return models.GameServer{}, err
	}

	if !a.available {
		return models.GameServer{}, matchmaking.ErrNoAvailableServers
	}

	return models.GameServer{
		Host: "203.0.113.10",
		Signaling: models.SignalingEndpoint{
			Protocol: "wss",
			Port:     32001,
			Path:     "/ws",
		},
		WebRTC: models.WebRTCEndpoint{
			Protocol: "udp",
			Port:     32002,
		},
	}, nil
}

var _ matchmaking.Allocator = (*MockAllocator)(nil)
