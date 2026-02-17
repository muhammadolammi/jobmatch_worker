package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
)

func processSession(ctx context.Context, workerConfig *WorkerConfig, sessionUUID uuid.UUID) error {
	session, err := workerConfig.DB.GetSession(ctx, sessionUUID)
	if err != nil {
		log.Println("Session not found:", err)
		return err
	}

	log.Println("Loaded session:", session.ID)
	if session.Status == "completed" {
		return nil
	}

	update := map[string]any{
		"session_id": session.ID,
		"status":     "processing",
		"message":    "analysis started",
		"timestamp":  time.Now(),
	}

	_ = publishSessionUpdate(workerConfig.RabbitChan, session.ID.String(), update)

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

		_ = publishSessionUpdate(workerConfig.RabbitChan, session.ID.String(), update)
		return err
	}

	workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
		Status: "completed",
		ID:     session.ID,
	})

	update["status"] = "completed"
	update["message"] = "analysis completed"
	update["timestamp"] = time.Now()

	_ = publishSessionUpdate(workerConfig.RabbitChan, session.ID.String(), update)
	return nil
}
func handleEvent(w http.ResponseWriter, r *http.Request, workerConfig *WorkerConfig) {
	ctx := r.Context()

	wrappedMessage := struct {
		Message struct {
			Data []byte `json:"data,omitempty"`
			ID   string `json:"id"`
		} `json:"message"`
		Subscription string `json:"subscription"`
	}{}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("io.ReadAll: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	// byte slice unmarshalling handles base64 decoding.
	if err := json.Unmarshal(body, &wrappedMessage); err != nil {
		log.Printf("json.Unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if wrappedMessage.Message.Data == nil {
		log.Println("Empty message data")
		http.Error(w, "empty message data", http.StatusBadRequest)
		return
	}

	type Payload struct {
		SessionID string `json:"session_id"`
	}

	var payload Payload
	if err := json.Unmarshal(wrappedMessage.Message.Data, &payload); err != nil {
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

	err = processSession(ctx, workerConfig, sessionUUID)
	if err != nil {
		log.Println("Session Processing failed sessionId:", sessionUUID, " error:", err)
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
