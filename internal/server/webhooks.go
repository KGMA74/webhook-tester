package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const maxRequests = 50

// WebhookRequest is an immutable snapshot of a captured HTTP request.
type WebhookRequest struct {
	ID        string              `json:"id"`
	Timestamp string              `json:"timestamp"`
	Method    string              `json:"method"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
}

// WebhookStore is a thread-safe, bounded ring-buffer of captured requests.
type WebhookStore struct {
	mu       sync.Mutex
	requests []WebhookRequest
}

func NewWebhookStore() *WebhookStore {
	return &WebhookStore{
		requests: make([]WebhookRequest, 0, maxRequests),
	}
}

// Add prepends req to the store and trims excess entries beyond maxRequests.
func (ws *WebhookStore) Add(req WebhookRequest) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.requests = append([]WebhookRequest{req}, ws.requests...)
	if len(ws.requests) > maxRequests {
		ws.requests = ws.requests[:maxRequests]
	}
}

// Clear removes all stored requests.
func (ws *WebhookStore) Clear() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.requests = ws.requests[:0]
}

// GetAll returns a copy of the current request list (newest first).
func (ws *WebhookStore) GetAll() []WebhookRequest {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	out := make([]WebhookRequest, len(ws.requests))
	copy(out, ws.requests)
	return out
}

// handleIndex renders the dashboard HTML with the current request history.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	requests := s.store.GetAll()
	data := struct {
		Requests []WebhookRequest
		Count    int
	}{
		Requests: requests,
		Count:    len(requests),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		log.Printf("template execute: %v", err)
	}
}

// handleWebhook captures any POST to /webhook into the in-memory store.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	s.store.Add(WebhookRequest{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Method:    r.Method,
		Headers:   map[string][]string(r.Header),
		Body:      formatBody(body),
	})
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"captured"}`)
}

// handleClear empties the request history and redirects back to the dashboard.
func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	s.store.Clear()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleAPIRequests streams the current request list as JSON for the polling client.
func (s *Server) handleAPIRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.store.GetAll()); err != nil {
		log.Printf("encode requests: %v", err)
	}
}

// formatBody pretty-prints JSON payloads; returns the raw string for all other content types.
func formatBody(b []byte) string {
	if len(b) == 0 {
		return "(empty body)"
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err == nil {
		return buf.String()
	}
	return string(b)
}
