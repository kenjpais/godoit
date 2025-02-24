package redishandler

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"os"
	"sync"
	"log"
)

var once sync.Once
var RedisInstance *RedisClient

type RedisClient struct {
	Rdb *redis.Client
	Ctx context.Context
}

func GetRedisClient() *RedisClient {
	once.Do(func() {
		if err := godotenv.Load("../../internal/config/.env"); err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
		// Read Redis configurations from environment variables
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "localhost" // fallback to localhost if not set
		}

		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379" // fallback to port 6379 if not set
		}

		log.Printf("Connecting to Redis at %s:%s", host, port)

		RedisInstance = &RedisClient{
			Rdb: redis.NewClient(&redis.Options{
				Addr: fmt.Sprintf("%s:%s", host, port),
			}),
			Ctx: context.Background(),
		}
	
		if RedisInstance.Rdb == nil {
			log.Fatal("Failed to initialize Redis client.")
		}

		_, err := RedisInstance.Rdb.Ping(RedisInstance.Ctx).Result()
		if err != nil {
			log.Fatalf("Failed to ping Redis: %v", err)
		}
	})

	return RedisInstance
}
