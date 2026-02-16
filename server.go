package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func server(workerConfig *WorkerConfig) {

	// Define CORS options
	corsOptions := cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8081", "https://jobmatch-backend-755404739186.us-east1.run.app"}, // You can customize this based on your needs

		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"client-api-key",
			"X-CSRF-Token",
			"x-paystack-signature",
			"Accept",
		},
		AllowCredentials: true,
		MaxAge:           300,
	}
	router := chi.NewRouter()
	apiRoute := chi.NewRouter()
	router.Use(cors.Handler(corsOptions))
	// ADD MIDDLREWARE
	// A good base middleware stack
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(workerConfig.ClientAuth())

	apiRoute.Get("/process-session", workerConfig.ProcessSession)

	router.Mount("/api", apiRoute)
	srv := &http.Server{
		Addr:              ":" + workerConfig.Port,
		Handler:           router,
		ReadHeaderTimeout: time.Minute,
	}

	log.Printf("Serving on port: %s\n", workerConfig.Port)
	log.Fatal(srv.ListenAndServe())
}
