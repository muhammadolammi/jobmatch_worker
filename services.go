package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/muhammadolammi/jobmatchworker/internal/database"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"

	"github.com/streadway/amqp"
)

func ConnectRabbit(ctx context.Context, cfg *WorkerConfig) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Println("RabbitMQ connect cancelled")
			return
		default:
		}

		conn, err := amqp.Dial(cfg.RABBITMQUrl)
		if err != nil {
			log.Println("RabbitMQ not ready:", err)
			sleepBackoff(&backoff, maxBackoff)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			log.Println("Rabbit channel error:", err)
			sleepBackoff(&backoff, maxBackoff)
			continue
		}

		// cfg.RabbitConn = conn
		cfg.RabbitChan = ch
		log.Println("✅ RabbitMQ connected")
		return
	}
}

func ConnectDB(ctx context.Context, cfg *WorkerConfig) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Println("DB connect cancelled")
			return
		default:
		}

		db, err := sql.Open("postgres", cfg.DBURL)
		if err != nil {
			log.Println("DB open error:", err)
			sleepBackoff(&backoff, maxBackoff)
			continue
		}

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(pingCtx)
		cancel()

		if err != nil {
			log.Println("DB not ready:", err)
			db.Close()
			sleepBackoff(&backoff, maxBackoff)
			continue
		}

		// Pool tuning (Cloud Run friendly)
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(5 * time.Minute)

		cfg.DB = database.New(db)
		log.Println("✅ Postgres connected")
		return
	}
}

func sleepBackoff(b *time.Duration, max time.Duration) {
	time.Sleep(*b)
	*b *= 2
	if *b > max {
		*b = max
	}
}
func LoadAWSConfig(cfg *WorkerConfig, r2Config *R2Config) {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2Config.AccessKey, r2Config.SecretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Println("error creating aws config", err)
		return
	}
	cfg.AwsConfig = &awsConfig
}

func LoadAgentRunner(cfg *WorkerConfig) {

	googleApiKey := os.Getenv("GOOGLE_API_KEY")
	if googleApiKey == "" {
		log.Println("empty GOOGLE_API_KEY in env, can't create agent")
		return
	}

	agentName := "resume analyzer"
	analyzer, err := GetAgent(googleApiKey, agentName)
	if err != nil {
		log.Printf("failed to create agent: %v\n", err)
		return
	}

	//  create session for ai use
	// Create a session to examine its properties.
	inMemoryService := session.InMemoryService()

	r, err := runner.New(runner.Config{
		AppName:        analyzer.Name(),
		Agent:          analyzer,
		SessionService: inMemoryService,
	})
	if err != nil {
		log.Fatalf("failed to create runner: %v", err)
	}
	cfg.AgentSessionService = inMemoryService
	cfg.AgentName = agentName
	cfg.AgentRunner = r
}
