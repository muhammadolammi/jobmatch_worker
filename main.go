package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	workerConfig := startUp()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleEvent(w, r, workerConfig)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Worker listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
