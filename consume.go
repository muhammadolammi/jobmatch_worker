package main

import (
	_ "github.com/lib/pq"
)

// func worker(id int, workerConfig *WorkerConfig, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	//    to consume message on the queue
// 	conn, err := amqp.Dial(workerConfig.RABBITMQUrl)
// 	if err != nil {
// 		log.Fatal("error dialling rabbitmq: " + err.Error())
// 	}
// 	defer conn.Close()

// 	ch, err := conn.Channel()
// 	if err != nil {
// 		log.Fatal("error connecting to rabbitmq channel: " + err.Error())
// 	}
// 	defer ch.Close()
// 	_, err = ch.QueueDeclare(
// 		"sessions", // queue name
// 		true,       // durable (survives broker restarts)
// 		false,      // auto-delete when unused
// 		false,      // exclusive
// 		false,      // no-wait
// 		nil,        // arguments
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to declare queue: %v", err)
// 	}

// 	msgs, err := ch.Consume(
// 		"sessions", // queue name
// 		"",         // consumer tag
// 		true,       // auto-ack
// 		false,      // exclusive
// 		false,      // no-local
// 		false,      // no-wait
// 		nil,        // arguments
// 	)
// 	if err != nil {
// 		log.Fatal("error consuming rabbitmq message: " + err.Error())
// 	}

// 	for msg := range msgs {
// 		// Unmarshal the body
// 		session := Session{}
// 		// log.Println(string(msg.Body))
// 		err = json.Unmarshal(msg.Body, &session)
// 		// log.Println(session)

// 		if err != nil {
// 			log.Printf("error unmarshalling message body. err: %v", err)
// 			// update session status as failed
// 			workerConfig.DB.UpdateSessionStatus(context.Background(), database.UpdateSessionStatusParams{
// 				Status: "failed",
// 				ID:     session.ID,
// 			})
// 			update := map[string]any{
// 				"session_id": session.ID,
// 				"status":     "failed",
// 				"message":    "analysis failed",
// 				"timestamp":  time.Now(),
// 			}
// 			err := publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
// 			if err != nil {
// 				log.Println("failed to publish update:", err)
// 			}

// 			continue
// 		}
// 		log.Printf("Worker %d processing session. session_id: %s", id+1, session.ID)

// 		update := map[string]any{
// 			"session_id": session.ID,
// 			"status":     "processing",
// 			"message":    "analysis started",
// 			"timestamp":  time.Now(),
// 		}
// 		err := publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
// 		if err != nil {
// 			log.Println("failed to publish update:", err)
// 		}
// 		workerConfig.DB.UpdateSessionStatus(context.Background(), database.UpdateSessionStatusParams{
// 			Status: "processing",
// 			ID:     session.ID,
// 		})

// 		err = callAgent(session, workerConfig)

// 		if err != nil {
// 			log.Printf("error running agent for session_id: %v. err: %v", session.ID, err)

// 			// update session status as failed
// 			workerConfig.DB.UpdateSessionStatus(context.Background(), database.UpdateSessionStatusParams{
// 				Status: "failed",
// 				ID:     session.ID,
// 			})
// 			update := map[string]any{
// 				"session_id": session.ID,
// 				"status":     "failed",
// 				"message":    "analysis failed",
// 				"timestamp":  time.Now(),
// 			}
// 			err := publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
// 			if err != nil {
// 				log.Println("failed to publish update:", err)
// 			}
// 			continue
// 		}
// 		// update session status

// 		workerConfig.DB.UpdateSessionStatus(context.Background(), database.UpdateSessionStatusParams{
// 			Status: "completed",
// 			ID:     session.ID,
// 		})
// 		update = map[string]any{
// 			"session_id": session.ID,
// 			"status":     "completed",
// 			"message":    "analysis completed",
// 			"timestamp":  time.Now(),
// 		}
// 		err = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
// 		if err != nil {
// 			log.Println("failed to publish update:", err)
// 		}
// 		// if err != nil {
// 		// 	log.Printf("error updating session status in db to completed for  session_id: %v. err: %v", session.ID, err)
// 		// 	continue
// 		// }
// 	}

// }

// func worker(workerConfig *WorkerConfig) {
// 	ctx := context.Background()

// 	client, err := pubsub.NewClient(ctx, workerConfig.ProjectID)
// 	if err != nil {
// 		log.Fatalf("Failed to create pubsub client: %v", err)
// 	}
// 	defer client.Close()

// 	sub := client.Subscriber("resume-analysis-sub")

// 	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {

// 		var payload struct {
// 			SessionID string `json:"session_id"`
// 		}

// 		if err := json.Unmarshal(msg.Data, &payload); err != nil {
// 			log.Println("Invalid message:", err)
// 			msg.Nack()
// 			return
// 		}

// 		sessionUUID, err := uuid.Parse(payload.SessionID)
// 		if err != nil {
// 			log.Println("Invalid UUID:", err)
// 			msg.Nack()
// 			return
// 		}

// 		// Load session from DB
// 		session, err := workerConfig.DB.GetSession(ctx, sessionUUID)
// 		if err != nil {
// 			log.Println("Session not found:", err)
// 			msg.Nack()
// 			return
// 		}

// 		log.Println("Processing session:", session.ID)

// 		// 🔵 1. Send "processing" update via Rabbit
// 		update := map[string]any{
// 			"session_id": session.ID,
// 			"status":     "processing",
// 			"message":    "analysis started",
// 			"timestamp":  time.Now(),
// 		}

// 		err = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)
// 		if err != nil {
// 			log.Println("failed to publish update:", err)
// 		}

// 		// Update DB
// 		workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
// 			Status: "processing",
// 			ID:     session.ID,
// 		})

// 		// 🔵 2. Run agent
// 		err = callAgent(dbSessionToSession(session), workerConfig)

// 		if err != nil {
// 			log.Printf("Agent failed for session %s: %v", session.ID, err)

// 			workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
// 				Status: "failed",
// 				ID:     session.ID,
// 			})

// 			update := map[string]any{
// 				"session_id": session.ID,
// 				"status":     "failed",
// 				"message":    "analysis failed",
// 				"timestamp":  time.Now(),
// 			}

// 			_ = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)

// 			msg.Ack() // acknowledge even if failed (job completed logically)
// 			return
// 		}

// 		// 🔵 3. Success
// 		workerConfig.DB.UpdateSessionStatus(ctx, database.UpdateSessionStatusParams{
// 			Status: "completed",
// 			ID:     session.ID,
// 		})

// 		update = map[string]any{
// 			"session_id": session.ID,
// 			"status":     "completed",
// 			"message":    "analysis completed",
// 			"timestamp":  time.Now(),
// 		}

// 		_ = publishSessionUpdate(workerConfig.RabbitConn, session.ID.String(), update)

// 		msg.Ack()
// 	})

// 	if err != nil {
// 		log.Fatalf("Receive error: %v", err)
// 	}
// }

// func (workerConfig *WorkerConfig) StartConsumerWorkerPool(numWorkers int) {
// 	var wg sync.WaitGroup
// 	wg.Add(numWorkers)

// 	for i := range numWorkers {
// 		log.Println("worker id ", i+1, "started")
// 		// wg.Done()
// 		// continue
// 		go worker(i, workerConfig, &wg)
// 	}
// 	wg.Wait() // block until all workers finish

// }

// func (workerConfig *WorkerConfig) StartWorker() {
// 	worker(workerConfig)
// }
