package executor

import (
	"container/heap"
	"doit/internal/api/middlewares"
	"doit/internal/db"
	"doit/internal/services/worker"
	"doit/pkg/utils"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"time"
)

const ScheduleQueryFreq = 1 * time.Minute

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) PrioritizeJobs(jobs []*db.Job) {
	pq := make(utils.PriorityQueue, len(jobs))
	for i, job := range jobs {
		pq[i] = &utils.JobItem{
			Value:    job,
			Priority: job.Priority,
		}
	}

	jobs = []*db.Job{}
	heap.Init(&pq)

	for pq.Len() > 0 {
		jobs = append(jobs, heap.Pop(&pq).(*utils.JobItem).Value)
	}
}

func (e *Executor) fetchSchedulesFromDB(maxRetries int) ([]db.Schedule, error) {
	var schedules []db.Schedule
	var err error
	retryCount := 0
	for retryCount < maxRetries {
		schedules, err = db.GetAllDueSchedules()
		if err == nil {
			return schedules, nil
		}
		log.Printf("Error fetching schedules from database: %v", err)
		if err := middlewares.ExponentialBackoff(retryCount+1, maxRetries); err != nil {
			return nil, err
		}
		retryCount++
	}
	
	return nil, err
}

func (e *Executor) Run(w *worker.WorkerPool) {
	if err := godotenv.Load("../../internal/config/.env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
		return
	}

	maxRetries, err := strconv.Atoi(os.Getenv("SCHEDULE_DB_MAX_RETRIES"))
	if err != nil {
		log.Fatalf("Error loading SCHEDULE_DB_MAX_RETRIES: %v", err)
		return
	}

	for {
		schedules, err := e.fetchSchedulesFromDB(maxRetries)
		if err != nil {
			log.Fatalf("Error fetching schedules after retries: %v", err)
			return
		}

		jobs := []*db.Job{}
		for _, schedule := range schedules {
			jobs = append(jobs, &db.Job{
				JobID:    schedule.JobID,
				Priority: schedule.Priority,
				Payload:  schedule.Payload,
			})
		}

		e.PrioritizeJobs(jobs)

		slotLength := len(jobs) / 3
		high := jobs[0:slotLength]
		mid := jobs[slotLength : 2*slotLength]
		low := jobs[2*slotLength:]

		e.distributeJobs(w, high, mid, low)

		time.Sleep(ScheduleQueryFreq)
	}
}

func (e *Executor) distributeJobs(w *worker.WorkerPool, high, mid, low []*db.Job) {
	for _, job := range high {
		w.HighChan <- job.JobID
	}
	for _, job := range mid {
		w.MidChan <- job.JobID
	}
	for _, job := range low {
		w.LowChan <- job.JobID
	}
}
