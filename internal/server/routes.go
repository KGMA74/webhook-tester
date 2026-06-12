package server

import "net/http"

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /new", s.handleNew)
	mux.HandleFunc("GET /health", s.handleHealth)

	// Webhook receiver — no method prefix = accepts every HTTP method
	mux.HandleFunc("/hook/{id}", s.handleWebhook)

	// Per-endpoint dashboard
	mux.HandleFunc("GET /{id}", s.requireAuth(s.handleIndex))

	// Internal API
	mux.HandleFunc("GET /api/{id}/events", s.requireAuth(s.handleSSE))
	mux.HandleFunc("GET /api/{id}/requests", s.requireAuth(s.handleAPIRequests))
	mux.HandleFunc("POST /api/{id}/clear", s.requireAuth(s.handleClear))
	mux.HandleFunc("POST /api/{id}/replay/{reqId}", s.requireAuth(s.handleReplay))
	mux.HandleFunc("GET /api/{id}/response-config", s.requireAuth(s.handleGetResponseConfig))
	mux.HandleFunc("PUT /api/{id}/response-config", s.requireAuth(s.handleSetResponseConfig))

	return mux
}
