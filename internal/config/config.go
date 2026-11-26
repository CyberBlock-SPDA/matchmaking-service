package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	AllocatorMode      string
	HTTPAddr           string
	KubeconfigPath     string
	GameNamespace      string
	GameFleetName      string
	AllocationTimeout  time.Duration
	JWTPrivateKeyPath  string
	TokenTTL           time.Duration
	SessionTTL         time.Duration
	CORSAllowedOrigins string
}

func Load() (Config, error) {
	cfg := Config{
		AllocatorMode:      getEnv("ALLOCATOR_MODE", "mock"),
		HTTPAddr:           getEnv("HTTP_ADDR", "127.0.0.1:8080"),
		KubeconfigPath:     os.Getenv("KUBECONFIG"),
		GameNamespace:      getEnv("GAME_NAMESPACE", "team-1"),
		GameFleetName:      getEnv("GAME_FLEET_NAME", "doom-fleet"),
		JWTPrivateKeyPath:  getEnv("JWT_PRIVATE_KEY_PATH", "keys/private.pem"),
		CORSAllowedOrigins: os.Getenv("CORS_ALLOWED_ORIGINS"),
	}

	if cfg.AllocatorMode != "mock" &&
		cfg.AllocatorMode != "agones" {
		return Config{}, fmt.Errorf(
			"invalid ALLOCATOR_MODE: %q",
			cfg.AllocatorMode,
		)
	}

	var err error

	cfg.AllocationTimeout, err = getDuration(
		"ALLOCATION_TIMEOUT", "5s",
	)
	if err != nil {
		return Config{}, err
	}

	cfg.TokenTTL, err = getDuration("TOKEN_TTL", "5m")
	if err != nil {
		return Config{}, err
	}

	cfg.SessionTTL, err = getDuration("SESSION_TTL", "10m")
	if err != nil {
		return Config{}, err
	}

	if cfg.TokenTTL > cfg.SessionTTL {
		return Config{}, fmt.Errorf(
			"TOKEN_TTL must not exceed SESSION_TTL",
		)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getDuration(key, defaultValue string) (time.Duration, error) {
	value := getEnv(key, defaultValue)

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return duration, nil
}
