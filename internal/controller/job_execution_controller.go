package controller

import (
	"encoding/json"
	"fmt"
	"doit/internal/db"
	redishandler "doit/internal/cache/redishandler"
	"doit/pkg/utils"
)

type JobExecutionController interface {
	CreateJobExecution(job *db.JobExecution) error
	GetJobExecution(jobId string) (*db.JobExecution, error)
	UpdateJobExecution(job *db.JobExecution) error
	DeleteJobExecution(jobId string) error
}

type JobExecutionOperationController struct{}

func NewJobExecutionOperationController() *JobExecutionOperationController {
	return &JobExecutionOperationController{}
}


func (jc *JobExecutionOperationController) CreateJobExecution(jobExec *db.JobExecution) error {
	jobExec.ProcessID = utils.GenerateProcessIDFromStruct(jobExec)

	if err := db.CreateJobExecution(jobExec); err != nil {
		return fmt.Errorf("failed to save job execution to database: %v", err)
	}

	jobExecJSON, err := json.Marshal(jobExec)
	if err != nil {
		return fmt.Errorf("failed to marshal JobExecution: %v", err)
	}

	rc := redishandler.GetRedisClient()
	if err := rc.Rdb.Set(rc.Ctx, "job:"+jobExec.JobID, jobExecJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to store JobExecution in Redis: %v", err)
	}

	return nil
}

func NewJobExecutionController(controllerType string) (*JobExecutionOperationController, error) {
	switch controllerType {
	case "JobExecutionOperationController":
		return NewJobExecutionOperationController(), nil
	default:
		return nil, fmt.Errorf("unknown controller type: %v", controllerType)
	}
}