package server

import "net/http"

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Dashboard — exact root only; {$} anchors the match to prevent catch-all behaviour.
	mux.HandleFunc("GET /{$}", s.handleIndex)

	// Capture endpoint — accept any webhook payload.
	mux.HandleFunc("POST /webhook", s.handleWebhook)

	// Clear the in-memory history.
	mux.HandleFunc("POST /clear", s.handleClear)

	// JSON feed consumed by the polling client for live updates.
	mux.HandleFunc("GET /api/requests", s.handleAPIRequests)

	return mux
}
