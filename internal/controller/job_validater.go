package controller

import (
	"fmt"
	"time"
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"
	mw "doit/internal/api/middlewares"
	"doit/pkg/utils"
	"doit/internal/db"
)

type JobValidator interface {
	ValidateJob(job *db.Job) error
	ValidateJobId(jobId string) error
}

type DefaultJobValidator struct{}

func (v *DefaultJobValidator) ValidateJob(job *db.Job) error {
	if err := mw.CheckJobIdempotency(job.JobID); err != nil {
		return fmt.Errorf("idempotency check failed: %v", err)
	}

	if job.TriggerAt.Before(time.Now()) {
		job.TriggerAt = time.Now()
	}

	if job.FinishAt.Before(time.Now()) {
		return fmt.Errorf("finishAt time [%v] is not ahead of current time", job.FinishAt)
	}

	if _, err := utils.EvalCronExpr(job.CronExpr); err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}

	return nil
}

func (v *DefaultJobValidator) ValidateJobId(jobId string) error {
	if !utils.ValidateId(jobId) {
		return fmt.Errorf("invalid job id")
	}
	return nil
}

type JobValidationController struct {
	joc       JobController
	validator JobValidator
}

func NewJobValidationController(joc JobController, validator JobValidator) *JobValidationController {
	return &JobValidationController{
		joc:       joc,
		validator: validator,
	}
}

func (jv *JobValidationController) CreateJob(job *db.Job) error {
	if err := jv.validator.ValidateJob(job); err != nil {
		return err
	}
	return jv.joc.CreateJob(job)
}

func (jv *JobValidationController) GetJob(jobId string) (*db.Job, error) {
	if err := jv.validator.ValidateJobId(jobId); err != nil {
		return nil, err
	}
	return jv.joc.GetJob(jobId)
}

func (jv *JobValidationController) UpdateJob(job *db.Job) error {
	if err := jv.validator.ValidateJobId(job.JobID); err != nil {
		return err
	}
	if err := jv.validator.ValidateJob(job); err != nil {
		return err
	}
	return jv.joc.UpdateJob(job)
}

func (jv *JobValidationController) DeleteJob(jobId string) error {
	if err := jv.validator.ValidateJobId(jobId); err != nil {
		return err
	}
	return jv.joc.DeleteJob(jobId)
}


// UploadJob validates the uploaded file (Python script) and checks its size and type
func (jv *JobValidationController) UploadJob(file *multipart.FileHeader, jobId string) error {
	if err := db.ValidateJobID(jobId); err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != db.AllowedFileExtension {
		return fmt.Errorf("invalid file type: only (%s) files are allowed", db.AllowedFileExtension)
	}
	if file.Size > db.MaxFileSize {
		return errors.New("file size exceeds the maximum limit")
	}
	
	return jv.joc.UploadJob(file, jobId)
}
