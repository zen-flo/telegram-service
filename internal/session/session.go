package session

import (
	"context"
	"github.com/zen-flo/telegram-service/internal/telegram"
	"sync/atomic"
	"time"
)

type Session struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc

	telegramClient *telegram.Client

	authReady atomic.Bool

	qrCode string
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

func (s *Session) QR() string {
	return s.qrCode
}

func (s *Session) MarkReady() {
	s.authReady.Store(true)
}

func (s *Session) IsReady() bool {
	return s.authReady.Load()
}

func (s *Session) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Session) Start() {
	if s.telegramClient == nil {
		return
	}
	go func() {
		_ = s.telegramClient.Start(s.ctx)
	}()
}

func (s *Session) StartQR(onReady func()) (string, error) {
	if s.telegramClient == nil {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()

	qr, err := s.telegramClient.StartQR(ctx, onReady)
	if err != nil {
		return "", err
	}

	s.qrCode = qr
	return qr, nil
}
