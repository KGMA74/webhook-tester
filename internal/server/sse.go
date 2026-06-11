package server

import (
	"fmt"
	"sync"
)

// SSEBroker distributes named Server-Sent Event frames to all connected clients.
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newSSEBroker() *SSEBroker {
	return &SSEBroker{clients: make(map[chan []byte]struct{})}
}

// subscribe registers a new client and returns its receive channel.
func (b *SSEBroker) subscribe() chan []byte {
	ch := make(chan []byte, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// unsubscribe removes the client and closes its channel.
func (b *SSEBroker) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// publish sends an SSE frame to every connected client.
// Slow clients are skipped without blocking.
func (b *SSEBroker) publish(event string, data []byte) {
	frame := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- frame:
		default:
		}
	}
}
