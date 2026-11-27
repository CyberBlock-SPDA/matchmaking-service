package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

type AdminHandler struct {
	terminator matchmaking.SessionTerminator
	token      string
}

func NewAdminHandler(
	terminator matchmaking.SessionTerminator,
	token string,
) *AdminHandler {
	return &AdminHandler{
		terminator: terminator,
		token:      token,
	}
}

func (h *AdminHandler) TerminateSession(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProblem(
			w,
			http.StatusMethodNotAllowed,
			models.CodeInvalidRequest,
			"Only POST method is allowed",
		)
		return
	}

	provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if h.token == "" ||
		provided == r.Header.Get("Authorization") ||
		subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeProblem(
			w,
			http.StatusUnauthorized,
			"ADMIN_UNAUTHORIZED",
			"Valid administrator credentials are required",
		)
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("sessionID"))
	if sessionID == "" {
		writeProblem(
			w,
			http.StatusBadRequest,
			models.CodeInvalidRequest,
			"Session ID is required",
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.terminator.TerminateSession(ctx, sessionID); err != nil {
		switch {
		case errors.Is(err, matchmaking.ErrSessionNotFound):
			writeProblem(
				w,
				http.StatusNotFound,
				"SESSION_NOT_FOUND",
				"No GameServer is associated with this session",
			)
		case errors.Is(err, matchmaking.ErrSessionNotActive):
			writeProblem(
				w,
				http.StatusConflict,
				"SESSION_NOT_ACTIVE",
				"The session is no longer active",
			)
		default:
			log.Printf("Session termination failed: %v", err)
			writeProblem(
				w,
				http.StatusInternalServerError,
				"SESSION_TERMINATION_FAILED",
				"Failed to terminate the game session",
			)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"sessionId": sessionID,
		"status":    "termination_requested",
	}); err != nil {
		log.Printf("Failed to write termination response: %v", err)
	}
}
