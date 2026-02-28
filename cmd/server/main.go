package main

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"

	"github.com/zen-flo/telegram-service/internal/app"
	"github.com/zen-flo/telegram-service/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize app", zap.Error(err))
	}

	if err := application.Run(ctx); err != nil {
		logger.Fatal("application stopped with error", zap.Error(err))
	}
}
