package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	// Parse CLI arguments
	addr := flag.String("addr", "0.0.0.0", "IP Address to bind to")
	port := flag.String("port", "8080", "Port to bind to")
	dir := flag.String("dir", ".", "Directory to serve files from")
	flag.Parse()

	// Validate directory
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Error resolving directory path: %v", err)
	}
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", absDir)
	}

	// Create custom file server handler
	fileServer := http.FileServer(http.Dir(absDir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			log.Printf("%s %s %d", r.Method, r.URL.Path, http.StatusMethodNotAllowed)
			return
		}

		// Create a custom ResponseWriter to capture the status code
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		fileServer.ServeHTTP(lrw, r)
		log.Printf("%s %s %d", r.Method, r.URL.Path, lrw.statusCode)
	})

	// Configure server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", *addr, *port),
		Handler: handler,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%s serving files from %s", *addr, *port, absDir)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for CTRL+C
	<-stop

	log.Println("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("Server stopped")
}

// loggingResponseWrite is a custom ResponseWriter that captures the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing it
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
