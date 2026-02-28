package session

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestManager_CreateAndDelete(t *testing.T) {
	manager := NewManager()

	s, err := manager.Create()
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	done := make(chan struct{})

	go func() {
		select {
		case <-s.Context().Done():
			close(done)
		}
	}()

	if err := manager.Delete(s.ID()); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	select {
	case <-done:
	// ok
	case <-time.After(time.Second):
		t.Fatal("expected session context to be cancelled")
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
			if _, err := manager.Create(); err != nil {
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

func TestManager_DeleteNotFound(t *testing.T) {
	manager := NewManager()

	err := manager.Delete("not-exist")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound")
	}
}
