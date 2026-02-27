package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrValidation = errors.New("configuration validation error")
	apiHashRegex  = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
)

type Config struct {
	GRPCPort        int
	TelegramAPIID   int
	TelegramAPIHash string
}

func Load() (*Config, error) {
	var validationErrors []string

	portStr := getEnv("GRPC_PORT", "50051")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		validationErrors = append(validationErrors, "GRPC_PORT must be a valid integer")
	} else if port < 1 || port > 65535 {
		validationErrors = append(validationErrors, "GRPC_PORT must be between 1 and 65535")
	}

	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	var apiID int
	if apiIDStr == "" {
		validationErrors = append(validationErrors, "TELEGRAM_API_ID is required")
	} else {
		apiID, err = strconv.Atoi(apiIDStr)
		if err != nil {
			validationErrors = append(validationErrors, "TELEGRAM_API_ID must be a valid integer")
		} else if apiID <= 0 {
			validationErrors = append(validationErrors, "TELEGRAM_API_ID must be positive")
		}
	}

	apiHash := os.Getenv("TELEGRAM_API_HASH")
	if apiHash == "" {
		validationErrors = append(validationErrors, "TELEGRAM_API_HASH is required")
	} else if !apiHashRegex.MatchString(apiHash) {
		validationErrors = append(validationErrors, "TELEGRAM_API_HASH must be a 32-character hex string")
	}

	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrValidation, strings.Join(validationErrors, "; "))
	}

	return &Config{
		GRPCPort:        port,
		TelegramAPIID:   apiID,
		TelegramAPIHash: apiHash,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
