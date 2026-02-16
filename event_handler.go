package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
)

func processSession(ctx context.Context, workerConfig *WorkerConfig, sessionUUID uuid.UUID) {
	session, err := workerConfig.DB.GetSession(ctx, sessionUUID)
	if err != nil {
		log.Println("Session not found:", err)
		return
	}

	log.Println("Loaded session:", session.ID)
	if session.Status == "completed" {
		return
	}

	update := map[string]any{
		"session_id": session.ID,
		"status":     "processing",
		"message":    "analysis started",
		"timestamp":  time.Now(),
	}

	_ = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)

	workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
		Status: "processing",
		ID:     session.ID,
	})

	err = callAgent(dbSessionToSession(session), workerConfig)

	if err != nil {
		log.Println("Agent failed:", err)

		workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
			Status: "failed",
			ID:     session.ID,
		})

		update["status"] = "failed"
		update["message"] = "analysis failed"
		update["timestamp"] = time.Now()

		_ = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
		return
	}

	workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
		Status: "completed",
		ID:     session.ID,
	})

	update["status"] = "completed"
	update["message"] = "analysis completed"
	update["timestamp"] = time.Now()

	_ = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
}

func handleEvent(w http.ResponseWriter, r *http.Request, workerConfig *WorkerConfig) {
	ctx := r.Context()

	var event struct {
		Data struct {
			Message struct {
				Data string `json:"data"`
			} `json:"message"`
		} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Println("Invalid event:", err)
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}

	// Decode base64 Pub/Sub payload
	msgBytes, err := base64.StdEncoding.DecodeString(event.Data.Message.Data)
	if err != nil {
		log.Println("Base64 decode failed:", err)
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}

	var payload struct {
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(msgBytes, &payload); err != nil {
		log.Println("Invalid payload:", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	sessionUUID, err := uuid.Parse(payload.SessionID)
	if err != nil {
		log.Println("Invalid UUID:", err)
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	log.Println("Processing session:", sessionUUID)

	processSession(ctx, workerConfig, sessionUUID)

	w.WriteHeader(http.StatusOK)
}
