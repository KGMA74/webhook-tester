package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"webhook-tester/cmd/web"
	"webhook-tester/internal/server"
)

func gracefulShutdown(srv *http.Server, done chan<- struct{}) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("shutdown signal received — draining connections (5s)")
	stop() // allow a second Ctrl-C to force exit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}

	log.Println("server exiting")
	close(done)
}

func main() {
	srv := server.NewServer(web.TemplateFS)

	done := make(chan struct{})
	go gracefulShutdown(srv, done)

	log.Printf("webhook-tester listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	<-done
	log.Println("graceful shutdown complete")
}
