package session

import (
	"crypto/rand"
	"errors"
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
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) Create() (*Session, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}

	session := New(id)

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

	s, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	s.Close()

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
