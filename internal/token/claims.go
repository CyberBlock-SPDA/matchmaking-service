package token

import "github.com/golang-jwt/jwt/v5"

const (
	Issuer   = "cloud-gaming-backend"
	Audience = "game-streamer"
)

type SessionClaims struct {
	GameID string `json:"gameId"`

	jwt.RegisteredClaims
}
