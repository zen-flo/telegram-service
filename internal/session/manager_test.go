package session

import (
	"errors"
	"sync"
	"testing"
)

func TestManager_CreateAndGet(t *testing.T) {
	manager := NewManager()

	s, err := manager.Create()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.ID() == "" {
		t.Fatal("expected non-empty session ID")
	}

	got, err := manager.Get(s.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ID() != s.ID() {
		t.Fatal("retrieved session does not match created one")
	}
}

func TestManager_Delete(t *testing.T) {
	manager := NewManager()

	s, err := manager.Create()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := manager.Delete(s.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = manager.Get(s.ID())
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestManager_ConcurrentCreate(t *testing.T) {
	manager := NewManager()

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := manager.Create()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	// check that all sessions are actually created
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if len(manager.sessions) != n {
		t.Fatalf("expected %d sessions, got %d", n, len(manager.sessions))
	}
}
