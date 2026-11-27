# matchmaking-service

A Go backend service for the CyberBlock cloud gaming platform. It creates game sessions, allocates game servers through Agones and issues Ed25519-signed JWT session tokens for game-streamer.

## Features

- Game session creation through REST API.
- Mock and Agones allocator modes.
- Session binding via GameServer annotations.
- Configurable CORS for client-web.
- Health and readiness checks, HTTP timeouts and graceful shutdown.

## Requirements

- Go 1.27.4 or newer.
- OpenSSL for local key generation.
- A Kubernetes cluster with Agones and a ready GameServer Fleet for real allocation.

## Local development

Run commands from the repository root. Generate a local Ed25519 key pair if you do not already have one:

```sh
mkdir -p keys
chmod 700 keys
openssl genpkey -algorithm ED25519 -out keys/private.pem
chmod 600 keys/private.pem
openssl pkey -in keys/private.pem -pubout -out keys/public.pem
openssl pkey -pubin -in keys/public.pem -outform DER | tail -c 32 | base64 -w 0 > keys/public.base64
```

Use `keys/public.base64` as `SESSION_PUBLIC_KEY` in game-streamer. Never commit private keys.

Run in mock mode (no Kubernetes required):

```sh
go run ./cmd/server
```

Run with Agones using a valid `KUBECONFIG`:

```sh
env ALLOCATOR_MODE=agones go run ./cmd/server
```

The server listens on `http://127.0.0.1:8080` by default.

## Configuration

| Variable | Default |
|---|---|
| `ALLOCATOR_MODE` | `mock` |
| `HTTP_ADDR` | `127.0.0.1:8080` |
| `KUBECONFIG` | Unset (in-cluster configuration) |
| `GAME_NAMESPACE` | `team-1` |
| `GAME_FLEET_NAME` | `doom-fleet` |
| `ALLOCATION_TIMEOUT` | `5s` |
| `JWT_PRIVATE_KEY_PATH` | `keys/private.pem` |
| `TOKEN_TTL` | `5m` |
| `SESSION_TTL` | `10m` |
| `CORS_ALLOWED_ORIGINS` | Unset (cross-origin access disabled) |

For local frontend development, set `CORS_ALLOWED_ORIGINS=http://localhost:5173`. Multiple origins can be separated by commas.

In Kubernetes, use `ALLOCATOR_MODE=agones`, `HTTP_ADDR=0.0.0.0:8080` and a private key mounted from a Secret. The ServiceAccount needs permission to create `gameserverallocations` in the game namespace. The Fleet must expose ports named `signaling` and `webrtc`.

The repository Dockerfile builds a non-root static container. Production
Deployment, ConfigMap, externally provisioned signing-key Secret and the
minimal Agones Role/RoleBinding are managed by
[`k8s-apps-infrastructure`](https://github.com/CyberBlock-SPDA/k8s-apps-infrastructure).

## API

| Method | Endpoint | Description |
|---|---|---|
| GET | `/healthz` | Liveness check |
| GET | `/readyz` | Application readiness check |
| POST | `/api/v1/sessions` | Create a game session |

Create a session:

```sh
curl -X POST http://127.0.0.1:8080/api/v1/sessions \
  -H 'Content-Type: application/json' \
  -d '{"gameId":"doom-1993"}'
```

The service returns HTTP 201 with the session ID, game ID, server endpoints, session token and expiration timestamps. If no server is available, it returns HTTP 503 with `Retry-After`. Errors use Problem Details.

The API contract is maintained in [api-spec](https://github.com/CyberBlock-SPDA/api-spec).

## Testing

```sh
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

Local integration tests with k3d and Agones verified successful allocation, server exhaustion, concurrent requests, session annotations, WSS authentication with the real game-streamer and Fleet recovery. HTTP handler and CORS tests also passed.

## MVP limitations

- Only `doom-1993` is supported.
- Session creation is not idempotent. Repeated client requests may allocate additional servers. Automatic allocation retries are prevented in the tested failure scenarios, but an uncertain allocation result may still leave an allocated server.
- Game-streamer and Agones manage server shutdown and replacement. The backend does not maintain a session registry or enforce `expiresAt` as a hard session termination deadline.
- Readiness checks application state, not Agones API availability.
- Full browser/WebRTC integration and cloud deployment remain to be verified.