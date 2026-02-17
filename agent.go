package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// retry retries a function up to `attempts` times with exponential backoff
func retry[T any](attempts int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for i := 0; i < attempts; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
		wait := time.Duration(500*(i+1)) * time.Millisecond
		time.Sleep(wait)
	}
	return zero, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

func aggregateResult(results *AnalysesResults, resultStr string, hasError bool, errorMsg string) {
	result := AnalysesResult{}
	switch {
	case hasError:
		result.IsErrorResult = true
		result.Error = errorMsg

	case strings.TrimSpace(resultStr) == "":
		result.IsErrorResult = true
		result.Error = "empty response from agent"

	default:
		cleaned := CleanJson(resultStr)

		if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
			result.IsErrorResult = true
			result.Error = "json unmarshal error: " + err.Error()
		}
	}

	results.Results = append(results.Results, result)
}

// callAgent runs the agent pipeline for all resumes in a given session.
// It handles downloading, text extraction, AI analysis, and DB persistence.
// Failures are retried selectively: network & DB retries only where needed.
func callAgent(currentSession Session, workerConfig *WorkerConfig) error {
	ctx := context.Background()
	// get resumes in session
	resumes, err := workerConfig.DB.GetResumesBySession(ctx, currentSession.ID)
	if err != nil {
		return fmt.Errorf("error getting resumes for session: %v, err: %v", currentSession.ID, err)
	}

	results := &AnalysesResults{
		SessionID: currentSession.ID,
	}

	// create an agent session
	agentSession, err := workerConfig.AgentSessionService.Create(ctx, &session.CreateRequest{
		AppName:   workerConfig.AgentName,
		UserID:    currentSession.UserID.String(),
		SessionID: currentSession.ID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}
	// process each resume
	for _, resume := range resumes {

		awsClient := s3.NewFromConfig(*workerConfig.AwsConfig, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", workerConfig.R2.AccountID))
		})

		// ✅ Retry downloading file (network failures are transient)
		fileBytes, err := retry(3, func() ([]byte, error) {
			return DownloadFromR2(ctx, awsClient, workerConfig.R2.Bucket, resume.ObjectKey)
		})
		if err != nil {
			log.Printf("⚠️ Failed to download %s after retries: %v", resume.ObjectKey, err)
			// aggregateResult(results, "", true, fmt.Sprintf("file download error: %v", err))
			aggregateResult(results, "", true, "file download error")

			continue
		}

		// Extract text from file
		resumeText, err := ExtractResumeText(resume.Mime, fileBytes)
		if err != nil {
			log.Printf("⚠️ Text extraction failed for %s: %v", resume.ObjectKey, err)
			aggregateResult(results, "", true, fmt.Sprintf("text extraction error: %v", err))
			continue
		}

		// Build AI input
		msg := fmt.Sprintf(
			"Job Title:\n%s\n\nJob Description:\n%s\n\nResume:\n%s",
			currentSession.JobTitle,
			currentSession.JobDescription,
			resumeText,
		)

		// ✅ Retry the AI agent stream separately (in case of transient agent failures)
		finalOutput, streamErr := retry(2,
			func() (string, error) {
				stream := workerConfig.AgentRunner.Run(ctx, agentSession.Session.UserID(), agentSession.Session.ID(), &genai.Content{
					Role: "user",
					Parts: []*genai.Part{
						{Text: msg},
					},
				}, agent.RunConfig{})

				var output string
				for event, err := range stream {
					if err != nil {
						return "", err
					}
					if event != nil && event.IsFinalResponse() && len(event.Content.Parts) > 0 {
						output = event.Content.Parts[0].Text
					}
				}

				if output == "" {
					return "", fmt.Errorf("empty agent response")
				}
				return output, nil
			})

		if streamErr != nil {
			log.Printf("⚠️ Agent failed for %s after retries: %v", resume.ObjectKey, streamErr)
			// log.Println("agent output: ", finalOutput)
			aggregateResult(results, "", true, fmt.Sprintf("agent stream error: %v", streamErr))
		} else {
			// log.Println("agent output: ", finalOutput)

			aggregateResult(results, finalOutput, false, "")
		}
	}
	log.Println("session id: " + agentSession.Session.ID() + " analyzed")
	// Clean up the session.
	err = workerConfig.AgentSessionService.Delete(ctx, &session.DeleteRequest{
		AppName:   agentSession.Session.AppName(),
		UserID:    agentSession.Session.UserID(),
		SessionID: agentSession.Session.ID(),
	})
	if err != nil {
		return fmt.Errorf("failed to delete session: %v", err)
	}

	// save final result to db
	resultsJSON, err := json.Marshal(results.Results)
	if err != nil {
		return fmt.Errorf("failed to marshal analyses results: %w", err)
	}

	_, err = retry(3, func() (any, error) {
		return nil, workerConfig.DB.CreateOrUpdateAnalysesResults(ctx, database.CreateOrUpdateAnalysesResultsParams{
			Results:   resultsJSON,
			SessionID: results.SessionID,
		})
	})
	if err != nil {
		return fmt.Errorf("failed to save agent result after retries: %w", err)
	}

	return nil
}
