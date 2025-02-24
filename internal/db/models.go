package db

import (
	"time"
)

const (
	// Maximum allowed file size (in bytes), for example 10MB
	MaxFileSize = 10 * 1024 * 1024
	AllowedFileExtension = ".py"
	ScriptPath = "../../scripts"
)

const (
	JobStatusCompleted = "completed"
	JobStatusRunning   = "running"
	JobStatusFailed    = "failed"
	JobStatusPending   = "pending"
	JobStatusCancelled = "cancelled"
)

const (
	WorkerActive   = "active"
	WorkerInactive = "inactive"
	WorkerPaused   = "paused"
)

type Job struct {
	JobID      string    `gorm:"primaryKey" json:"job_id"`
	UserID     string    `json:"user_id"`
	CronExpr   string    `json:"cron"`
	Priority   int       `json:"priority"`
	Payload    string    `json:"payload"`
	Type       string    `json:"type"`
	MaxRetries int       `json:"max_retries"`
	RcreTime   time.Time `json:"rcre_time"`
	TriggerAt  time.Time `json:"trigger_at"`
	FinishAt   time.Time `json:"finish_at"`
}

type JobExecution struct {
	ProcessID string    `gorm:"primaryKey" json:"process_id"`
	JobID     string    `json:"job_id"`
	WorkerID  string    `json:"worker_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
	Error     string    `json:"error"`
}

type Schedule struct {
	JobID       string    `gorm:"primaryKey" json:"job_id"`
	Priority    int       `json:"priority"`
	Payload     string    `json:"payload"`
	RetryCount  int       `json:"retry_count"`
	MaxRetries  int       `json:"max_retries"`
	ExecTime    time.Time `json:"exec_time"`
	Duration    int       `json:"duration"`
	RcreTime    time.Time `json:"rcre_time"`
	NextRunTime time.Time `json:"next_run_time"`
	LastRunTime time.Time `json:"last_run_time"`
}

type Worker struct {
	WorkerID      string    `gorm:"primaryKey" json:"worker_id"`
	IPAddress     string    `json:"ip_address"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Capacity      int       `json:"capacity"`
	CurrentLoad   int       `json:"current_load"`
}
