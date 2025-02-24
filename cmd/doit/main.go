package main

import (
	"doit/internal/animation"
	"doit/internal/api"
	"doit/internal/db"
	"doit/internal/services/executor"
	"doit/internal/services/scheduler"
	"doit/internal/services/worker"
	redishandler "doit/internal/cache/redishandler"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	err := godotenv.Load("../../internal/config/.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	animation.StartBootAnimation()

	if err := db.ConnectDatabase(); err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	rc := redishandler.GetRedisClient()

	if err := rc.Rdb.Set(rc.Ctx, "job:"+"1212", "something", 0).Err(); err != nil {
		log.Printf("failed to store Job in Redis: %v", err)
	} else {
		log.Println("Successfully stored job in Redis")
	}

	s := scheduler.NewScheduler()
	e := executor.NewExecutor()
	w := worker.NewWorkerPool()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		s.Run()
	}()
	go func() {
		defer wg.Done()
		e.Run(w)
	}()
	go func() {
		defer wg.Done()
		w.Run()
	}()

	// Graceful shutdown handling using signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go api.StartServer()

	// Wait for termination signal
	<-shutdown
	log.Println("Shutting down gracefully...")

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("Shutdown complete.")
}
