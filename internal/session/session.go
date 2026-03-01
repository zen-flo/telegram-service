package session

import (
	"context"
	"errors"
	"github.com/zen-flo/telegram-service/internal/broker"
	"github.com/zen-flo/telegram-service/internal/telegram"
	"sync/atomic"
	"time"
)

type TelegramClient interface {
	Start(ctx context.Context) error
	StartQR(ctx context.Context, onReady func()) (string, error)
	SendMessage(ctx context.Context, peer, text string) (int64, error)
	Messages() <-chan *telegram.IncomingMessage
}

type Session struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc

	telegramClient *telegram.Client
	dispatcher     *broker.Dispatcher

	authReady atomic.Bool
	qrCode    string
}

func New(id string, client *telegram.Client, dispatcher *broker.Dispatcher) *Session {
	ctx, cancel := context.WithCancel(context.Background())

	return &Session{
		id:             id,
		ctx:            ctx,
		cancel:         cancel,
		telegramClient: client,
		dispatcher:     dispatcher,
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

func (s *Session) SubscribeMessages() <-chan *broker.Message {
	return s.dispatcher.Subscribe(s.id)
}

func (s *Session) Unsubscribe(ch chan *broker.Message) {
	s.dispatcher.Unsubscribe(s.id, ch)
}

func (s *Session) SendMessage(peer, text string) (int64, error) {
	if !s.IsReady() {
		return 0, errors.New("session not authorized")
	}

	return s.telegramClient.SendMessage(
		s.ctx,
		peer,
		text,
	)
}
