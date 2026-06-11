package server

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Endpoint holds the isolated state for a single webhook listener.
type Endpoint struct {
	ID        string
	CreatedAt time.Time
	store     *WebhookStore
	broker    *SSEBroker
}

// EndpointRegistry is a thread-safe map of live endpoints.
type EndpointRegistry struct {
	mu    sync.RWMutex
	items map[string]*Endpoint
}

func newEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{items: make(map[string]*Endpoint)}
}

// Create generates a new Endpoint with a random 16-char hex ID and registers it.
func (r *EndpointRegistry) Create() *Endpoint {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	ep := &Endpoint{
		ID:        hex.EncodeToString(b),
		CreatedAt: time.Now(),
		store:     NewWebhookStore(),
		broker:    newSSEBroker(),
	}
	r.mu.Lock()
	r.items[ep.ID] = ep
	r.mu.Unlock()
	return ep
}

// Get returns the endpoint for the given ID, or false if it doesn't exist.
func (r *EndpointRegistry) Get(id string) (*Endpoint, bool) {
	r.mu.RLock()
	ep, ok := r.items[id]
	r.mu.RUnlock()
	return ep, ok
}
