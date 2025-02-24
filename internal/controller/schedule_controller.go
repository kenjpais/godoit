package controller

import (
	"encoding/json"
	"fmt"
	"time"
	"doit/internal/db"
	redishandler "doit/internal/cache/redishandler"
	"github.com/go-redis/redis/v8"
)

type ScheduleController interface {
	CreateSchedule(schedule *db.Schedule) error
	GetSchedule(scheduleID string) (*db.Schedule, error)
	UpdateSchedule(schedule *db.Schedule) error
	DeleteSchedule(jobID string) error
}

type ScheduleOperationController struct {
	redisClient *redis.Client
}

func NewScheduleOperationController(redisClient *redis.Client) *ScheduleOperationController {
	return &ScheduleOperationController{redisClient: redisClient}
}

func (sc *ScheduleOperationController) cacheScheduleInRedis(schedule *db.Schedule) error {
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %v", err)
	}

	// Set to Redis with a TTL (optional, e.g., 1 hour)
	ttl := 1 * time.Hour
	if err := sc.redisClient.Set(sc.redisClient.Context(), "JobSchedule:"+schedule.JobID, scheduleJSON, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store schedule in Redis: %v", err)
	}
	return nil
}

func (sc *ScheduleOperationController) getScheduleFromCacheOrDB(scheduleID string) (*db.Schedule, error) {
	scheduleJSON, err := sc.redisClient.Get(sc.redisClient.Context(), "Schedule:"+scheduleID).Result()
	if err == redis.Nil {
		schedule, err := db.GetSchedule(scheduleID)
		if err != nil {
			return nil, fmt.Errorf("schedule not found in database: %v", err)
		}
		if err := sc.cacheScheduleInRedis(&schedule); err != nil {
			return nil, fmt.Errorf("failed to cache schedule in Redis: %v", err)
		}
		return &schedule, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule from Redis: %v", err)
	}

	var schedule db.Schedule
	if err := json.Unmarshal([]byte(scheduleJSON), &schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule data: %v", err)
	}

	return &schedule, nil
}

func (sc *ScheduleOperationController) CreateSchedule(schedule *db.Schedule) error {
	schedule.RcreTime = time.Now()
	if err := db.CreateSchedule(schedule); err != nil {
		return fmt.Errorf("failed to save schedule to database: %v", err)
	}

	if err := sc.cacheScheduleInRedis(schedule); err != nil {
		return err
	}

	return nil
}

func (sc *ScheduleOperationController) GetSchedule(scheduleID string) (*db.Schedule, error) {
	schedule, err := sc.getScheduleFromCacheOrDB(scheduleID)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

func (sc *ScheduleOperationController) UpdateSchedule(schedule *db.Schedule) error {
	if err := db.UpdateSchedule(*schedule); err != nil {
		return fmt.Errorf("failed to update schedule in database: %v", err)
	}

	if err := sc.cacheScheduleInRedis(schedule); err != nil {
		return err
	}

	return nil
}

func (sc *ScheduleOperationController) DeleteSchedule(jobID string) error {
	if err := db.DeleteSchedule(jobID); err != nil {
		return fmt.Errorf("failed to delete schedule from database: %v", err)
	}

	if err := sc.redisClient.Del(sc.redisClient.Context(), "JobSchedule:"+jobID).Err(); err != nil {
		return fmt.Errorf("failed to delete schedule from Redis: %v", err)
	}

	return nil
}

func CreateScheduleController(controllerType string) (ScheduleController, error) {
	switch controllerType {
	case "ScheduleOperationController":
		return NewScheduleOperationController(redishandler.GetRedisClient().Rdb), nil
	default:
		return nil, fmt.Errorf("unknown controller type: %v", controllerType)
	}
}
