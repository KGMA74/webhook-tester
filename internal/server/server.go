package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port  int
	store *WebhookStore
	tmpl  *template.Template
}

func NewServer(templateFS fs.FS) *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080
	}

	tmpl, err := template.New("index.html").
		Funcs(template.FuncMap{"join": strings.Join}).
		ParseFS(templateFS, "template/index.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	srv := &Server{
		port:  port,
		store: NewWebhookStore(),
		tmpl:  tmpl,
	}

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.port),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}
