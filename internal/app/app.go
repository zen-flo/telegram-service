package app

import (
	"context"

	"github.com/zen-flo/telegram-service/internal/config"
	"github.com/zen-flo/telegram-service/internal/grpc"
	"go.uber.org/zap"
)

type App struct {
	server *grpc.Server
}

func New(
	cfg *config.Config,
	logger *zap.Logger,
) (*App, error) {

	telegramHandler := grpc.NewTelegramHandler()

	server := grpc.NewServer(
		cfg.GRPCPort,
		logger,
		telegramHandler,
	)

	return &App{
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.server.Start(ctx)
}
