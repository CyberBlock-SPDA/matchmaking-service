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
env ALLOCATOR_MODE=agones MATCHMAKING_ADMIN_TOKEN="$(openssl rand -hex 32)" go run ./cmd/server
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
| `MATCHMAKING_ADMIN_TOKEN` | Required when `ALLOCATOR_MODE=agones`; administrator Bearer secret |
| `ALLOCATION_TIMEOUT` | `5s` |
| `JWT_PRIVATE_KEY_PATH` | `keys/private.pem` |
| `TOKEN_TTL` | `5m` |
| `SESSION_TTL` | `10m` |
| `CORS_ALLOWED_ORIGINS` | Unset (cross-origin access disabled) |

For local frontend development, set `CORS_ALLOWED_ORIGINS=http://localhost:5173`. Multiple origins can be separated by commas.

In Kubernetes, use `ALLOCATOR_MODE=agones`, `HTTP_ADDR=0.0.0.0:8080`, a private key mounted from a Secret, and `MATCHMAKING_ADMIN_TOKEN` from a separate Secret. The ServiceAccount needs permission to create `gameserverallocations` and list/delete GameServers in the game namespace. The Fleet must expose ports named `signaling` and `webrtc`.

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
| GET | `/api/v1/ice-credentials` | Issue short-lived TURN REST credentials when TURN is configured |
| POST | `/api/v1/admin/sessions/{sessionId}/terminate` | Terminate an allocated session (admin Bearer token required) |

Create a session:

```sh
curl -X POST http://127.0.0.1:8080/api/v1/sessions \
  -H 'Content-Type: application/json' \
  -d '{"gameId":"doom-1993"}'
```

The service returns HTTP 201 with the session ID, game ID, server endpoints, session token and expiration timestamps. If no server is available, it returns HTTP 503 with `Retry-After`. Errors use Problem Details.

Terminate a session using its returned `sessionId`:

```sh
curl --cacert university-root-ca.pem \
  -X POST \
  -H "Authorization: Bearer $(cat ~/.local/share/cyberblock-pki/matchmaking-admin-token)" \
  "https://<traefik-lan-ip>/api/v1/admin/sessions/<session-id>/terminate"
```

The endpoint returns `202` when termination has been requested. Agones deletes
the allocated GameServer and the Fleet replaces it. `404` means no GameServer
is associated with the session; `409` means it is no longer allocated. The
backend remains stateless: it reads the session annotation from the Agones
GameServer rather than maintaining a registry. `MATCHMAKING_ADMIN_TOKEN` must
be a long random value supplied through a Kubernetes Secret and is required
when `ALLOCATOR_MODE=agones`.

Configure TURN with `TURN_URLS` (comma-separated `turn:`/`turns:` URLs) and
`TURN_SHARED_SECRET` (at least 32 bytes). When configured together, the
`/api/v1/ice-credentials` endpoint returns coturn-compatible HMAC-SHA1
credentials with an expiry embedded in the username. Credentials are
short-lived and responses are marked `Cache-Control: no-store`; keep the
shared secret in an external Kubernetes Secret, never in the frontend or
ConfigMap. `TURN_CREDENTIAL_TTL` defaults to ten minutes and cannot exceed one
hour.

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
- Game-streamer and Agones manage normal server shutdown and replacement. The admin endpoint can terminate an allocated session, but the backend does not maintain a session registry or enforce `expiresAt` as a hard session termination deadline.
- Readiness checks application state, not Agones API availability.
- Full browser/WebRTC integration and cloud deployment remain to be verified.