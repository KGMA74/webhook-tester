package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port     int
	registry *EndpointRegistry
	tmpls    *template.Template
	token    string // empty = auth disabled
}

func NewServer(templateFS fs.FS) *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080
	}

	tmpls, err := template.New("").ParseFS(templateFS, "template/*.html")
	if err != nil {
		slog.Error("failed to parse templates", "err", err)
		os.Exit(1)
	}

	srv := &Server{
		port:     port,
		registry: newEndpointRegistry(),
		tmpls:    tmpls,
		token:    os.Getenv("WEBHOOK_TOKEN"),
	}

	if srv.token != "" {
		slog.Info("auth enabled — UI protected with WEBHOOK_TOKEN")
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", srv.port),
		Handler: srv.RegisterRoutes(),
		// WriteTimeout must be 0 to allow SSE connections to stay open indefinitely.
		WriteTimeout: 0,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}
}
