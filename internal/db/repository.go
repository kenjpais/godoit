package db

import (
	"database/sql"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"io"
	"time"
	"path/filepath"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	_ "github.com/lib/pq"
)

var DB *gorm.DB

// ensureDatabaseExists checks if the database exists, and creates it if not
func ensureDatabaseExists() error {
	// Connect to PostgreSQL without specifying a database
	adminDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_SSL"),
	)

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}
	defer adminDB.Close()

	// Check if the target database exists
	dbName := os.Getenv("DB_NAME")
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = '%s')", dbName)
	err = adminDB.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %v", err)
	}

	// If the database does not exist, create it
	if !exists {
		_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			return fmt.Errorf("failed to create database: %v", err)
		}
		log.Printf("Database %s created successfully!", dbName)
	} else {
		log.Printf("Database %s already exists.", dbName)
	}

	return nil
}

// ConnectDatabase initializes the database connection
func ConnectDatabase() error {
	// Ensure the database exists before connecting
	err := ensureDatabaseExists()
	if err != nil {
		log.Fatalf("Database check failed: %v", err)
		return err
	}

	// Connect to the specified database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSL"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return err
	}

	// Migrate the schemas
	err = db.AutoMigrate(&Job{}, &JobExecution{}, &Schedule{}, &Worker{})
	if err != nil {
		log.Fatalf("Failed to migrate database schemas: %v", err)
		return err
	}

	DB = db
	fmt.Println("Database connected and schemas migrated!")
	return nil
}

func CreateJob(job Job) error {
	if err := DB.Create(&job).Error; err != nil {
		return err
	}
	return nil
}

func GetJob(jobID string) (Job, error) {
	var job Job
	if err := DB.First(&job, "job_id = ?", jobID).Error; err != nil {
		return Job{}, err
	}
	return job, nil
}

func GetAllJobs(limit, offset int) ([]Job, error) {
	var jobs []Job
	if err := DB.Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func UpdateJob(job Job) error {
	if err := DB.Save(&job).Error; err != nil {
		return err
	}
	return nil
}

func DeleteJob(jobID string) error {
	if err := DB.Delete(&Job{}, "job_id = ?", jobID).Error; err != nil {
		return err
	}
	return nil
}

func CreateSchedule(Schedule *Schedule) error {
	if err := DB.Create(&Schedule).Error; err != nil {
		return err
	}
	return nil
}

func GetSchedule(jobId string) (Schedule, error) {
	var schedule Schedule
	if err := DB.First(&schedule, "job_id = ?", jobId).Error; err != nil {
		return Schedule{}, err
	}
	return schedule, nil
}

func GetAllSchedules(limit, offset int) ([]Schedule, error) {
	var Schedules []Schedule
	if err := DB.Limit(limit).Offset(offset).Find(&Schedules).Error; err != nil {
		return nil, err
	}
	return Schedules, nil
}

func GetAllDueSchedules() ([]Schedule, error) {
	var schedules []Schedule
	if err := DB.
		Where("next_run_time <= ?", time.Now()).
		Find(&schedules).Error; err != nil {
		return nil, err
	}

	return schedules, nil
}

func UpdateSchedule(Schedule Schedule) error {
	if err := DB.Save(&Schedule).Error; err != nil {
		return err
	}
	return nil
}

func DeleteSchedule(JobID string) error {
	if err := DB.Delete(&Schedule{}, "job_id = ?", JobID).Error; err != nil {
		return err
	}
	return nil
}

func CreateJobExecution(je *JobExecution) error {
	if err := DB.Create(&je).Error; err != nil {
		return err
	}
	return nil
}

func ValidateJobID(jobId string) error {
	var job Job
	if err := DB.First(&job, "job_id = ?", jobId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("jobId does not exist")
		}
		return err
	}

	return nil
}

func UpdateJobPayload(jobID string, newPayload string) error {
	var job Job

	if err := DB.First(&job, "job_id = ?", jobID).Error; err != nil {
		return fmt.Errorf("job not found: %v", err)
	}

	job.Payload = newPayload
	if err := DB.Save(&job).Error; err != nil {
		return fmt.Errorf("failed to update job payload: %v", err)
	}

	return nil
}

func SaveJobScript(file *multipart.FileHeader) error {
	uploadDir := ScriptPath
	err := os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create upload directory: %v", err)
	}

	filePath := filepath.Join(uploadDir, file.Filename)  

	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file at destination: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to save uploaded file: %v", err)
	}
	
	return nil
}

func (j *JobExecution) BeforeSave(tx *gorm.DB) (err error) {
	if j.Status != JobStatusPending && j.Status != JobStatusRunning && j.Status != JobStatusCompleted && j.Status != JobStatusFailed && j.Status != JobStatusCancelled {
		return fmt.Errorf("invalid job status: %s", j.Status)
	}

	return nil
}

func (w *Worker) BeforeSave(tx *gorm.DB) (err error) {
	if w.Status != WorkerActive && w.Status != WorkerInactive && w.Status != WorkerPaused {
		return fmt.Errorf("invalid worker status: %s", w.Status)
	}
	if w.CurrentLoad < 0 {
		return fmt.Errorf("current load cannot be negative")
	}
	if w.CurrentLoad > w.Capacity {
		return fmt.Errorf("current load cannot exceed capacity")
	}
	return nil
}
