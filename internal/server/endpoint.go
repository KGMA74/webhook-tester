package server

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// ResponseConfig holds the custom HTTP response an endpoint returns to callers.
type ResponseConfig struct {
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

// Endpoint holds the isolated state for a single webhook listener.
type Endpoint struct {
	ID        string
	CreatedAt time.Time
	store     *WebhookStore
	broker    *SSEBroker
	mu        sync.RWMutex
	respCfg   *ResponseConfig
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

// GetResponseConfig returns the current response config, falling back to the default.
func (ep *Endpoint) GetResponseConfig() ResponseConfig {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	if ep.respCfg == nil {
		return ResponseConfig{StatusCode: 200, ContentType: "application/json", Body: `{"status":"captured"}`}
	}
	return *ep.respCfg
}

// SetResponseConfig replaces the endpoint's response config.
func (ep *Endpoint) SetResponseConfig(cfg ResponseConfig) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.respCfg = &cfg
}
