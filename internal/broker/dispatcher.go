package broker

import (
	"sync"
)

type Message struct {
	ID        int64
	From      string
	Text      string
	Timestamp int64
}

type Dispatcher struct {
	mu   sync.RWMutex
	subs map[string]map[chan *Message]struct{}
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		subs: make(map[string]map[chan *Message]struct{}),
	}
}

func (d *Dispatcher) Subscribe(sessionID string) <-chan *Message {
	ch := make(chan *Message, 16)

	d.mu.Lock()
	if _, ok := d.subs[sessionID]; !ok {
		d.subs[sessionID] = make(map[chan *Message]struct{})
	}
	d.subs[sessionID][ch] = struct{}{}
	d.mu.Unlock()

	return ch
}

func (d *Dispatcher) Publish(sessionID string, msg *Message) {
	d.mu.RLock()
	subs := d.subs[sessionID]
	d.mu.RUnlock()

	for ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (d *Dispatcher) Unsubscribe(sessionID string, ch chan *Message) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if subs, ok := d.subs[sessionID]; ok {
		delete(subs, ch)
		close(ch)
	}
}
