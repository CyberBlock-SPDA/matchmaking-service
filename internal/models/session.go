package models

import "time"

type CreateSessionRequest struct {
	GameID string `json:"gameId"`
}

type CreateSessionResponse struct {
	SessionID      string     `json:"sessionId"`
	GameID         string     `json:"gameId"`
	Server         GameServer `json:"server"`
	SessionToken   string     `json:"sessionToken"`
	TokenExpiresAt time.Time  `json:"tokenExpiresAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
}

type GameServer struct {
	Host      string            `json:"host"`
	Signaling SignalingEndpoint `json:"signaling"`
	WebRTC    WebRTCEndpoint    `json:"webrtc"`
}

type SignalingEndpoint struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Path     string `json:"path"`
}

type WebRTCEndpoint struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}
