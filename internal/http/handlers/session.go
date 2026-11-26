package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/models"
)

const (
	supportedGameID = "doom-1993"
	maxRequestSize  = 1024
)

type SessionHandler struct {
	service *matchmaking.Service
}

func NewSessionHandler(service *matchmaking.Service) *SessionHandler {
	return &SessionHandler{
		service: service,
	}
}

func (h *SessionHandler) CreateSession(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)

		writeProblem(w, http.StatusMethodNotAllowed,
			models.CodeInvalidRequest,
			"Only POST method is allowed")
		return
	}

	mediaType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)

	if err != nil || mediaType != "application/json" {
		writeProblem(w, http.StatusBadRequest,
			models.CodeInvalidRequest,
			"Content-Type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	var request models.CreateSessionRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeProblem(w, http.StatusBadRequest,
			models.CodeInvalidRequest,
			"Invalid request body")
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeProblem(w, http.StatusBadRequest,
			models.CodeInvalidRequest,
			"Request must contain exactly one JSON object")
		return
	}

	if request.GameID != supportedGameID {
		writeProblem(w, http.StatusBadRequest,
			models.CodeInvalidRequest,
			"Unsupported gameId")
		return
	}

	response, err := h.service.CreateSession(
		r.Context(),
		request.GameID,
	)

	if err != nil {
		handleSessionError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to write session response: %v", err)
	}
}

func handleSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, matchmaking.ErrNoAvailableServers):
		w.Header().Set("Retry-After", "5")
		writeProblem(w, http.StatusServiceUnavailable,
			models.CodeNoServers,
			"No game servers are currently available. Please try again later.")

	case errors.Is(err, matchmaking.ErrAllocationFailed):
		log.Printf("Allocation failed: %v", err)

		writeProblem(w, http.StatusInternalServerError,
			models.CodeAllocationFailed,
			"Failed to allocate a game server")

	default:
		log.Printf("Session creation failed: %v", err)

		writeProblem(w, http.StatusInternalServerError,
			models.CodeInternalError,
			"An internal server error occurred")
	}
}

func writeProblem(
	w http.ResponseWriter,
	status int,
	code string,
	detail string,
) {
	problem := models.Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Code:   code,
		Detail: detail,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		log.Printf("Failed to write problem response: %v", err)
	}
}
