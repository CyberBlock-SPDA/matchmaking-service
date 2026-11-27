package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/turncredentials"
)

type TURNHandler struct {
	credentials *turncredentials.Service
}

func NewTURNHandler(credentials *turncredentials.Service) *TURNHandler {
	return &TURNHandler{credentials: credentials}
}

func (h *TURNHandler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	credentials, err := h.credentials.Credentials()
	if err != nil {
		log.Printf("Failed to generate TURN credentials: %v", err)
		http.Error(w, "failed to generate ICE credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(credentials); err != nil {
		log.Printf("Failed to write TURN credentials: %v", err)
	}
}
