# Matchmaking Service

A Go backend service for the cloud gaming platform responsible for creating game sessions, allocating game servers through Agones, and issuing signed JWT session tokens.

## Current functionality

- REST API for game session creation.
- Game server allocation through Agones and Kubernetes API.
- Ed25519 JWT session token signing.
- Mock and Agones allocator modes.
- Environment-based configuration.
- Health check endpoint.
- Unit tests and local integration testing.

## API

| Method | Endpoint | Description |
|---|---|---|
| GET | `/healthz` | Service health check |
| POST | `/api/v1/sessions` | Create a game session |

## Local development

Requirements: Go, an Ed25519 private key, and a Kubernetes cluster with Agones for real allocation.

Run in mock mode:

`go run ./cmd/server`

Run with Agones:

`env ALLOCATOR_MODE=agones go run ./cmd/server`

By default, the HTTP server listens on `127.0.0.1:8080`.

## Testing

`go test ./...`

`go vet ./...`

## Project status

MVP in development. Successful allocation, server exhaustion, and concurrent allocation have been verified in a local Kubernetes cluster with Agones.

Game-streamer integration and production deployment are pending.