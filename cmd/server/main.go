package main

import (
	"log"
	"net/http"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/allocator"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/config"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/http/handlers"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/kubernetes"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/matchmaking"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/token"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	var gameAllocator matchmaking.Allocator

	switch cfg.AllocatorMode {
	case "mock":
		gameAllocator = allocator.NewMockAllocator(true)

	case "agones":
		client, err := kubernetes.NewClient(cfg.KubeconfigPath)
		if err != nil {
			log.Fatalf("create Kubernetes client: %v", err)
		}

		gameAllocator = allocator.NewAgonesAllocator(
			client,
			cfg.GameNamespace,
			cfg.GameFleetName,
			cfg.AllocationTimeout,
		)

	default:
		log.Fatalf("unsupported allocator mode: %s", cfg.AllocatorMode)
	}

	tokenSigner, err := token.NewSigner(cfg.JWTPrivateKeyPath)
	if err != nil {
		log.Fatalf("create token signer: %v", err)
	}

	service := matchmaking.NewService(
		gameAllocator,
		tokenSigner,
		cfg.TokenTTL,
		cfg.SessionTTL,
	)

	sessionHandler := handlers.NewSessionHandler(service)

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc(
		"/api/v1/sessions",
		sessionHandler.CreateSession,
	)

	log.Printf("allocator mode: %s", cfg.AllocatorMode)

	if cfg.AllocatorMode == "agones" {
		log.Printf(
			"Agones namespace: %s, fleet: %s",
			cfg.GameNamespace,
			cfg.GameFleetName,
		)
	}

	log.Printf("HTTP server listening on %s", cfg.HTTPAddr)

	if err := http.ListenAndServe(cfg.HTTPAddr, mux); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}
