package session

import (
	"context"
	"errors"
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

	msgSubs chan chan *Message
}

func New(id string, client *telegram.Client) *Session {
	ctx, cancel := context.WithCancel(context.Background())

	return &Session{
		id:             id,
		ctx:            ctx,
		cancel:         cancel,
		telegramClient: client,
		msgSubs:        make(chan chan *Message, 16),
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

func (s *Session) SubscribeMessages() <-chan *Message {
	ch := make(chan *Message, 16)

	select {
	case s.msgSubs <- ch:
	default:
	}

	return ch
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
