package app

import (
	"context"
	"github.com/zen-flo/telegram-service/internal/session"

	"github.com/zen-flo/telegram-service/internal/config"
	"github.com/zen-flo/telegram-service/internal/grpc"
	"go.uber.org/zap"
)

type App struct {
	cfg    *config.Config
	logger *zap.Logger
	server *grpc.Server
}

func New(
	cfg *config.Config,
	logger *zap.Logger,
) (*App, error) {

	sessionManager := session.NewManager(logger)

	telegramHandler := grpc.NewTelegramHandler(
		sessionManager,
		logger,
	)

	server := grpc.NewServer(
		cfg.GRPCPort,
		logger,
		telegramHandler,
	)

	return &App{
		cfg:    cfg,
		logger: logger,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.server.Start(ctx)
}
