package session

import (
	"crypto/rand"
	"errors"
	"github.com/zen-flo/telegram-service/internal/telegram"
	"go.uber.org/zap"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	logger   *zap.Logger
}

func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger:   logger,
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) Create() (*Session, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}

	tgClient := telegram.NewClient(m.logger)

	session := New(id, tgClient)

	// start client lifecycle inside session runtime
	go func() {
		if err := tgClient.Start(session.Context()); err != nil {
			m.logger.Error("telegram client stopped", zap.Error(err))
		}
	}()

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	session.Close()

	delete(m.sessions, id)
	return nil
}

func generateID() (string, error) {
	t := time.Now().UTC()
	entropy := ulid.Monotonic(rand.Reader, 0)

	id, err := ulid.New(ulid.Timestamp(t), entropy)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
