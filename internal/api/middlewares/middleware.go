package middlewares

import (
	"doit/internal/cache/redishandler"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"time"
)

var rateLimiter = rate.NewLimiter(rate.Every(time.Minute), 100) // 100 requests per minute

func RateLimitMiddleware(c *gin.Context) {
	if !rateLimiter.Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
		c.Abort()
		return
	}
	c.Next()
}

func CheckJobIdempotency(jobId string) error {
	rc := redishandler.GetRedisClient()
	if exists, _ := rc.Rdb.Exists(rc.Ctx, "job:"+jobId).Result(); exists == 1 {
		return fmt.Errorf("job already exists")
	}
	return nil
}

func ExponentialBackoff(retryCount int, maxRetries int) error {
    baseDelay := time.Second  // Initial backoff delay
    maxDelay := time.Minute  // Maximum delay between retries

    if retryCount >= maxRetries {
        return fmt.Errorf("failed to connect to database after %d retries", maxRetries)
    }

    // Exponential backoff: 2^retryCount backoff
    delay := time.Duration(int64(baseDelay) * int64(1<<retryCount))
    if delay > maxDelay {
        delay = maxDelay
    }

    log.Printf("Retrying in %v... (attempt %d/%d)", delay, retryCount+1, maxRetries)
    time.Sleep(delay)
	
    return nil
}

