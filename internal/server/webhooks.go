package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const maxRequests = 50

// WebhookRequest is an immutable snapshot of one captured HTTP request.
type WebhookRequest struct {
	ID        string              `json:"id"`
	Timestamp string              `json:"timestamp"`
	Method    string              `json:"method"`
	Query     map[string][]string `json:"query,omitempty"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
}

// WebhookStore is a thread-safe, bounded in-memory store (newest-first).
type WebhookStore struct {
	mu       sync.Mutex
	requests []WebhookRequest
}

func NewWebhookStore() *WebhookStore {
	return &WebhookStore{requests: make([]WebhookRequest, 0, maxRequests)}
}

func (ws *WebhookStore) Add(req WebhookRequest) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.requests = append([]WebhookRequest{req}, ws.requests...)
	if len(ws.requests) > maxRequests {
		ws.requests = ws.requests[:maxRequests]
	}
}

func (ws *WebhookStore) Clear() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.requests = ws.requests[:0]
}

func (ws *WebhookStore) GetAll() []WebhookRequest {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	out := make([]WebhookRequest, len(ws.requests))
	copy(out, ws.requests)
	return out
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

// handleHome serves the landing page.
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpls.ExecuteTemplate(w, "home.html", nil); err != nil {
		slog.Error("template execute home", "err", err)
	}
}

// handleNew creates a fresh endpoint and redirects the browser to its dashboard.
func (s *Server) handleNew(w http.ResponseWriter, r *http.Request) {
	ep := s.registry.Create()
	slog.Info("endpoint created", "id", ep.ID)
	http.Redirect(w, r, "/"+ep.ID, http.StatusSeeOther)
}

// handleIndex serves the per-endpoint dashboard shell.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.registry.Get(r.PathValue("id")); !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpls.ExecuteTemplate(w, "index.html", nil); err != nil {
		slog.Error("template execute index", "err", err)
	}
}

// handleSSE opens a persistent event stream scoped to one endpoint.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	ep, ok := s.registry.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := ep.broker.subscribe()
	defer ep.broker.unsubscribe(ch)

	// Push current state immediately so the client renders on (re)connect.
	data, _ := json.Marshal(ep.store.GetAll())
	fmt.Fprintf(w, "event: init\ndata: %s\n\n", data)
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg) //nolint:errcheck
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleWebhook captures any incoming request on /hook/{id} and broadcasts it.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ep, ok := s.registry.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	req := WebhookRequest{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Method:    r.Method,
		Headers:   map[string][]string(r.Header),
		Body:      formatBody(body),
	}
	if q := r.URL.Query(); len(q) > 0 {
		req.Query = map[string][]string(q)
	}

	ep.store.Add(req)

	data, _ := json.Marshal(req)
	ep.broker.publish("webhook", data)

	slog.Info("webhook captured",
		"endpoint", ep.ID,
		"method", req.Method,
		"content-type", r.Header.Get("Content-Type"),
		"bytes", len(body),
	)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"captured"}`)
}

// handleClear empties the endpoint store and notifies SSE clients.
func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	ep, ok := s.registry.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	ep.store.Clear()
	ep.broker.publish("cleared", []byte("{}"))
	slog.Info("history cleared", "endpoint", ep.ID)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"cleared"}`)
}

// handleAPIRequests returns the current request list as JSON.
func (s *Server) handleAPIRequests(w http.ResponseWriter, r *http.Request) {
	ep, ok := s.registry.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ep.store.GetAll()); err != nil {
		slog.Error("encode requests", "err", err)
	}
}

// requireAuth wraps a handler with Basic-Auth enforcement when WEBHOOK_TOKEN is set.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" {
			next(w, r)
			return
		}
		_, pass, ok := r.BasicAuth()
		if ok && pass == s.token {
			next(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="webhook-tester"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

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
