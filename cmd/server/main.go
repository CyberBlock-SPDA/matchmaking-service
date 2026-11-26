package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/CyberBlock-SPDA/matchmaking-service/internal/allocator"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/config"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/http/handlers"
	"github.com/CyberBlock-SPDA/matchmaking-service/internal/http/middleware"
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

	var ready atomic.Bool

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	mux.HandleFunc(
		"/api/v1/sessions",
		sessionHandler.CreateSession,
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           middleware.CORS(cfg.CORSAllowedOrigins, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      cfg.AllocationTimeout + 10*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Printf("allocator mode: %s", cfg.AllocatorMode)

	if cfg.AllocatorMode == "agones" {
		log.Printf(
			"Agones namespace: %s, fleet: %s",
			cfg.GameNamespace,
			cfg.GameFleetName,
		)
	}

	serverErrors := make(chan error, 1)

	ready.Store(true)

	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		ready.Store(false)

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}

	case <-ctx.Done():
		ready.Store(false)
		stop()

		log.Println("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			cfg.AllocationTimeout+10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)

			if err := server.Close(); err != nil {
				log.Printf("force close server: %v", err)
			}

			os.Exit(1)
		}

		log.Println("HTTP server stopped gracefully")
	}
}
