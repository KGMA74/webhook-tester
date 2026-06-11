package server

import "net/http"

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Landing page
	mux.HandleFunc("GET /{$}", s.handleHome)

	// Create a new endpoint and redirect to its dashboard
	mux.HandleFunc("GET /new", s.handleNew)

	// Health check — always open
	mux.HandleFunc("GET /health", s.handleHealth)

	// Webhook receiver — no method prefix = accepts every HTTP method
	mux.HandleFunc("/hook/{id}", s.handleWebhook)

	// Per-endpoint dashboard (auth-gated when WEBHOOK_TOKEN is set).
	// Internal API routes use /api/ prefix to avoid wildcard conflicts with /hook/{id}.
	mux.HandleFunc("GET /{id}", s.requireAuth(s.handleIndex))
	mux.HandleFunc("GET /api/{id}/events", s.requireAuth(s.handleSSE))
	mux.HandleFunc("GET /api/{id}/requests", s.requireAuth(s.handleAPIRequests))
	mux.HandleFunc("POST /api/{id}/clear", s.requireAuth(s.handleClear))

	return mux
}
