package main

import (
	"context"
	"log"
	"os"
)

func startUp() *WorkerConfig {
	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Println("empty DB_URL in environment")
	}
	projectId := os.Getenv("PROJECT_ID")

	if projectId == "" {
		log.Println("empty PROJECT_ID in environment")
	}

	rabbitmqUrl := os.Getenv("RABBITMQ_URL")
	if rabbitmqUrl == "" {
		log.Println("empty RABBITMQ_URL in env")
	}

	r2AccountId := os.Getenv("R2_ACCOUNT_ID")
	if r2AccountId == "" {
		log.Println("empty R2_ACCOUNT_ID in environment")
	}
	r2Bucket := os.Getenv("R2_BUCKET")
	if r2Bucket == "" {
		log.Println("empty R2_BUCKET in environment")
	}
	r2SecretKey := os.Getenv("R2_SECRET_KEY")
	if r2SecretKey == "" {
		log.Println("empty R2_SECRET_KEY in environment")
	}
	r2AccessKey := os.Getenv("R2_ACCESS_KEY")
	if r2AccessKey == "" {
		log.Println("empty R2_ACCESS_KEY in environment")
	}
	r2Config := R2Config{
		AccountID: r2AccountId,
		AccessKey: r2AccessKey,
		SecretKey: r2SecretKey,
		Bucket:    r2Bucket,
	}
	// awsConfig, err := config.LoadDefaultConfig(context.TODO(),
	// 	config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2Config.AccessKey, r2Config.SecretKey, "")),
	// 	config.WithRegion("auto"),
	// )
	// if err != nil {
	// 	log.Fatal("error creating aws config", err)
	// }

	// create agent and runner

	// conn, err := amqp.Dial(rabbitmqUrl)
	// if err != nil {
	// 	log.Fatalf("error connecting to RabbitMQ. err:  %v", err)

	// }
	//  update config agent runner.
	workerConfig := WorkerConfig{
		DBURL: dbUrl,
		// GoogleApiKey:        googleApiKey,
		R2: &r2Config,
		// AwsConfig:   &awsConfig,
		RABBITMQUrl: rabbitmqUrl,
	}
	ctx := context.Background()
	go LoadAWSConfig(&workerConfig, &r2Config)
	go ConnectDB(ctx, &workerConfig)
	go ConnectRabbit(ctx, &workerConfig)
	go LoadAgentRunner(&workerConfig)
	return &workerConfig
}
