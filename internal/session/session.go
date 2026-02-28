package session

import (
	"context"
	"github.com/zen-flo/telegram-service/internal/telegram"
)

type Session struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc

	telegramClient *telegram.Client
}

func New(id string, client *telegram.Client) *Session {
	ctx, cancel := context.WithCancel(context.Background())

	return &Session{
		id:             id,
		ctx:            ctx,
		cancel:         cancel,
		telegramClient: client,
	}
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Context() context.Context {
	return s.ctx
}

func (s *Session) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
