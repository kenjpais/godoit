package controller

import (
	redishandler "doit/internal/cache/redishandler"
	"doit/internal/db"
	"doit/pkg/utils"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"time"
	"github.com/go-redis/redis/v8"
)

type JobController interface {
	CreateJob(job *db.Job) error
	GetJob(jobId string) (*db.Job, error)
	UpdateJob(job *db.Job) error
	DeleteJob(jobId string) error
	UploadJob(*multipart.FileHeader, string) error
}

type JobOperationController struct{}

func NewJobOperationController() *JobOperationController {
	return &JobOperationController{}
}


func (jc *JobOperationController) CreateJob(job *db.Job) error {
	job.RcreTime = time.Now()
	job.JobID = utils.GenerateJobIDFromStruct(job)

	if err := db.CreateJob(*job); err != nil {
		return fmt.Errorf("failed to save job to database: %v", err)
	}

	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal Job: %v", err)
	}

	rc := redishandler.GetRedisClient()
	if err := rc.Rdb.Set(rc.Ctx, "job:"+job.JobID, jobJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to store Job in Redis: %v", err)
	}

	return nil
}

func (jc *JobOperationController) GetJob(jobID string) (*db.Job, error) {
	rc := redishandler.GetRedisClient()
	jobJSON, err := rc.Rdb.Get(rc.Ctx, "job:"+jobID).Result()
	if err == redis.Nil {
		job, err := db.GetJob(jobID)
		if err != nil {
			return nil, fmt.Errorf("job not found")
		}
		jobJSON, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal job data")
		}
		rc.Rdb.Set(rc.Ctx, "job:"+job.JobID, string(jobJSON), 0)
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch job from Redis")
	}

	var job db.Job
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job data")
	}

	return &job, nil
}

func (jc *JobOperationController) UploadJob(file *multipart.FileHeader, jobId string) error {
	if err := db.SaveJobScript(file); err != nil {
		return err
	}
	payload := "doit/scripts/" + file.Filename
	if err := db.UpdateJobPayload(jobId, payload); err != nil {
		return fmt.Errorf("UpdateJobPayload failed: %v", err)
	}

	return nil
}

func (jc *JobOperationController) UpdateJob(job *db.Job) error {
	if err := db.UpdateJob(*job); err != nil {
		return err
	}

	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job data")
	}

	rc := redishandler.GetRedisClient()
	if err := rc.Rdb.Set(rc.Ctx, "job:"+job.JobID, jobJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to store updated job in Redis: %v", err)
	}

	return nil
}

func (jc *JobOperationController) DeleteJob(jobID string) error {
	if err := db.DeleteJob(jobID); err != nil {
		return err
	}

	rc := redishandler.GetRedisClient()
	if err := rc.Rdb.Del(rc.Ctx, "job:"+jobID).Err(); err != nil {
		return fmt.Errorf("failed to delete job from Redis: %v", err)
	}

	return nil
}

func NewJobController(controllerType string) (JobController, error) {
	switch controllerType {
	case "JobOperationController":
		return NewJobValidationController(NewJobOperationController(), &DefaultJobValidator{}), nil
	default:
		return nil, fmt.Errorf("unknown controller type: %v", controllerType)
	}
}