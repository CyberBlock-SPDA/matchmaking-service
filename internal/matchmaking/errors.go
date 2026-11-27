package matchmaking

import "errors"

var (
	ErrNoAvailableServers = errors.New("no available game servers")
	ErrAllocationFailed   = errors.New("game server allocation failed")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionNotActive   = errors.New("session is not active")
)
