package telegram

import (
	"context"

	"github.com/gotd/td/telegram"
	"go.uber.org/zap"
)

type Client struct {
	client *telegram.Client
	logger *zap.Logger
}

func NewClient(logger *zap.Logger) *Client {
	return &Client{
		logger: logger,
	}
}

func (c *Client) Start(ctx context.Context) error {
	// for now, it's just a placeholder lifecycle
	<-ctx.Done()
	return nil
}
