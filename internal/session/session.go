package session

import "context"

type Session struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc
}

func New(id string) *Session {
	ctx, cancel := context.WithCancel(context.Background())

	return &Session{
		id:     id,
		ctx:    ctx,
		cancel: cancel,
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
