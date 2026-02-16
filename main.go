package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/joho/godotenv"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
	"github.com/streadway/amqp"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

func main() {
	_ = godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("empty DB_URL in environment")
	}

	rabbitmqUrl := os.Getenv("RABBITMQ_URL")
	if rabbitmqUrl == "" {
		log.Fatal("empty RABBITMQ_URL in env")
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("error opening db. err: ", err)
	}

	dbqueries := database.New(db)

	r2AccountId := os.Getenv("R2_ACCCOUNT_ID")
	if r2AccountId == "" {
		log.Fatal("empty R2_ACCCOUNT_ID in environment")
	}
	r2Bucket := os.Getenv("R2_BUCKET")
	if r2Bucket == "" {
		log.Fatal("empty R2_BUCKET in environment")
	}
	r2SecretKey := os.Getenv("R2_SECRET_KEY")
	if r2SecretKey == "" {
		log.Fatal("empty R2_SECRET_KEY in environment")
	}
	r2AccessKey := os.Getenv("R2_ACCESS_KEY")
	if r2AccessKey == "" {
		log.Fatal("empty R2_ACCESS_KEY in environment")
	}
	r2Config := R2Config{
		AccountID: r2AccountId,
		AccessKey: r2AccessKey,
		SecretKey: r2SecretKey,
		Bucket:    r2Bucket,
	}
	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2Config.AccessKey, r2Config.SecretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal("error creating aws config", err)
	}

	googleApiKey := os.Getenv("GOOGLE_API_KEY")
	if googleApiKey == "" {
		log.Fatal("empty GOOGLE_API_KEY in env")
	}
	// create agent and runner
	agentName := "resume analyzer"
	analyzer, err := GetAgent(googleApiKey, agentName)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	//  create session for ai use
	// Create a session to examine its properties.
	inMemoryService := session.InMemoryService()

	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:        analyzer.Name(),
		Agent:          analyzer,
		SessionService: inMemoryService,
	})
	if err != nil {
		log.Fatalf("failed to create runner: %v", err)
	}
	conn, err := amqp.Dial(rabbitmqUrl)
	if err != nil {
		log.Fatalf("error connecting to RabbitMQ. err:  %v", err)

	}
	//  update config agent runner.
	workerConfig := WorkerConfig{
		AgentName:           agentName,
		AgentRunner:         r,
		AgentSessionService: inMemoryService,
		DB:                  dbqueries,
		// GoogleApiKey:        googleApiKey,
		R2:          &r2Config,
		AwsConfig:   &awsConfig,
		RABBITMQUrl: rabbitmqUrl,
		RabbitConn:  conn,
	}

	fmt.Println("Starting 3 workers consumer pool ")
	workerConfig.StartConsumerWorkerPool(3)
}
