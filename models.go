package main

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
	"github.com/streadway/amqp"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

type R2Config struct {
	AccountID string
	Bucket    string
	AccessKey string
	SecretKey string
}

type WorkerConfig struct {
	DB *database.Queries
	// GoogleApiKey        string
	R2                  *R2Config
	AwsConfig           *aws.Config
	RabbitConn          *amqp.Connection
	RABBITMQUrl         string
	AgentRunner         *runner.Runner
	AgentSessionService session.Service
	AgentName           string
}

type AnalysesResult struct {
	CandidateEmail      string   `json:"candidate_email"`
	MatchScore          int      `json:"match_score"`
	RelevantExperiences []string `json:"relevant_experiences"`
	RelevantSkills      []string `json:"relevant_skills"`
	MissingSkills       []string `json:"missing_skills"`
	Summary             string   `json:"summary"`
	Recomendation       string   `json:"recommendation"`
	// Error result entry
	IsErrorResult bool   `json:"is_error_result"`
	Error         string `json:"error,omitempty"`
}
type AnalysesResults struct {
	ID        uuid.UUID        `json:"id"`
	Results   []AnalysesResult `json:"results" db:"results"`
	CreatedAt time.Time        `json:"created_at"`
	SessionID uuid.UUID        `json:"session_id"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type Session struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	Name           string    `json:"name"`
	UserID         uuid.UUID `json:"user_id"`
	Status         string    `json:"status"`
	JobTitle       string    `json:"job_title"`
	JobDescription string    `json:"job_description"`
}
