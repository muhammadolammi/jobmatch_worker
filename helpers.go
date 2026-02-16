package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ledongthuc/pdf"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
	"github.com/nguyenthenguyen/docx"
	"github.com/streadway/amqp"
)

func CleanJson(input string) string {
	clean := strings.TrimSpace(input)

	// Remove opening ```json or ``` with optional newline
	if strings.HasPrefix(clean, "```json") {
		clean = strings.TrimPrefix(clean, "```json")
	} else if strings.HasPrefix(clean, "```") {
		clean = strings.TrimPrefix(clean, "```")
	}
	clean = strings.TrimLeft(clean, "\r\n") // remove newline immediately after opening backticks

	// Remove closing ``` unconditionally
	clean = strings.TrimSuffix(clean, "```")

	clean = strings.TrimSpace(clean) // final trim

	return clean

}

// --- File Download ---

func DownloadFromR2(ctx context.Context, client *s3.Client, bucket, key string) ([]byte, error) {
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer out.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}
	return buf.Bytes(), nil
}

func ExtractResumeText(mime string, data []byte) (string, error) {
	switch mime {
	case "text/plain":
		return string(data), nil

	case "application/pdf":
		return extractPDFText(bytes.NewReader(data))

	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return extractDocxText(bytes.NewReader(data))

	default:
		return "", fmt.Errorf("unsupported file type: %s", mime)
	}
}

func extractPDFText(reader io.ReaderAt) (string, error) {
	pdfReader, err := pdf.NewReader(reader, int64(lenReader(reader)))
	if err != nil {
		return "", fmt.Errorf("failed to read pdf: %w", err)
	}
	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, _ := page.GetPlainText(nil)
		textBuilder.WriteString(text)
	}
	return textBuilder.String(), nil
}

func extractDocxText(reader io.Reader) (string, error) {
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, reader)
	if err != nil {
		return "", err
	}
	r := bytes.NewReader(buf.Bytes())

	doc, err := docx.ReadDocxFromMemory(r, int64(buf.Len()))
	if err != nil {
		return "", fmt.Errorf("failed to parse docx: %w", err)
	}
	defer doc.Close()

	return doc.Editable().GetContent(), nil
}

// Utility: get reader length for PDF
func lenReader(r io.ReaderAt) int64 {
	switch v := r.(type) {
	case *bytes.Reader:
		return int64(v.Len())
	default:
		return 0
	}
}

func publishSessionUpdate(rabbitConn *amqp.Connection, sessionID string, update map[string]any) error {
	ch, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	body, _ := json.Marshal(update)
	routingKey := fmt.Sprintf("session.%s", sessionID)

	return ch.Publish(
		"session_updates", // exchange
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func RespondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshalling payload to data %v", err)
		w.WriteHeader(500)
	}
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		log.Printf("error writing data to response %v", err)
		w.WriteHeader(500)
	}
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJson(w, code, map[string]string{"error": message})

}

func processSession(session Session, config WorkerConfig) {
	ctx := context.Background()
	sessionIDStr := session.ID.String()

	// 1. Mark as Processing
	log.Printf("Processing session_id: %s", sessionIDStr)
	config.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
		Status: "processing",
		ID:     session.ID,
	})

	update := map[string]any{
		"session_id": session.ID,
		"status":     "processing",
		"message":    "analysis started",
		"timestamp":  time.Now(),
	}
	// We use the existing RabbitConn in your config to send SSE updates
	publishSessionUpdate(config.RabbitConn, sessionIDStr, update)

	// 2. Call the Agent/Analyzer
	err := callAgent(session, &config)
	if err != nil {
		log.Printf("error running agent for session_id: %v. err: %v", session.ID, err)

		config.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
			Status: "failed",
			ID:     session.ID,
		})

		publishSessionUpdate(config.RabbitConn, sessionIDStr, map[string]any{
			"session_id": session.ID,
			"status":     "failed",
			"message":    "analysis failed",
			"timestamp":  time.Now(),
		})
		return
	}

	// 3. Mark as Completed
	config.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
		Status: "completed",
		ID:     session.ID,
	})

	publishSessionUpdate(config.RabbitConn, sessionIDStr, map[string]any{
		"session_id": session.ID,
		"status":     "completed",
		"message":    "analysis completed",
		"timestamp":  time.Now(),
	})
}
