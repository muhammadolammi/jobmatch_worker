package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	workerConfig := startUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleEvent(w, r, workerConfig)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Run server in goroutine
	go func() {
		log.Printf("Worker listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v\n", err)
		}
	}()

	// Listen for shutdown signal (Cloud Run sends SIGTERM)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)

	<-stop
	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new requests
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v\n", err)
	}

	// Now close resources safely
	if workerConfig.DBConn != nil {
		workerConfig.DBConn.Close()
		log.Println("DB connection closed")
	}

	if workerConfig.RabbitChan != nil {
		workerConfig.RabbitChan.Close()
		log.Println("Rabbit channel closed")
	}

	if workerConfig.RabbitConn != nil {
		workerConfig.RabbitConn.Close()
		log.Println("Rabbit connection closed")
	}

	log.Println("Shutdown complete")
}
