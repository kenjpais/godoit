package worker

import (
	"doit/internal/controller"
	"doit/internal/db"
	"doit/pkg/utils"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const MaxWorkers = 9

const HighCap = 70
const MidCap = 20
const LowCap = 10

const HighRefillRate = 70 * time.Millisecond
const MidRefillRate = 20 * time.Millisecond
const LowRefillRate = 10 * time.Millisecond

var (
	highBucket = utils.NewTokenBucket(HighCap, HighRefillRate)
	midBucket  = utils.NewTokenBucket(MidCap, MidRefillRate)
	lowBucket  = utils.NewTokenBucket(LowCap, LowRefillRate)
)

type Worker struct {
	Id string
}

type WorkerPool struct {
	pool     *sync.Pool
	HighChan chan string
	MidChan  chan string
	LowChan  chan string
	wg       sync.WaitGroup
	size     int
}

func NewWorkerPool() *WorkerPool {
	wp := &WorkerPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return &Worker{}
			},
		},
		HighChan: make(chan string),
		MidChan:  make(chan string),
		LowChan:  make(chan string),
		size:     MaxWorkers,
	}
	return wp
}

func dummyJob(jobId string, workerId string) {
	log.Printf("\nJob %s started on worker %v", jobId, workerId)
	time.Sleep(10 * time.Second)
	log.Printf("\nJob %s ended on worker %v", jobId, workerId)
}

func errorJob(jobId string, workerId string) error {
	log.Printf("\nJob %s started on worker %v", jobId, workerId)
	time.Sleep(10 * time.Second)
	return errors.New("job error")
}

func (w *Worker)Start(jobId string) error {
	// Assuming scriptPath is given as "../doit/scripts/123.py"
	scriptPath := "../doit/scripts/123.py"

	paths := strings.Split(scriptPath, "/")
	dir := strings.Join(paths[:len(paths)-1], "/")
	pythonScript := paths[len(paths)-1]

	// Optionally, log the paths to check if everything is correct
	log.Printf("Changing to directory: %s", dir)
	log.Printf("Running Python script: %s", pythonScript)

	// Change to the script directory
	err := os.Chdir(dir)
	if err != nil {
		return errors.New("Error changing directory: " + err.Error())
	}

	// Install dependencies and build image (for your comment placeholder)
	// You can implement dependency installation logic here if needed

	// Create a command to run the Python script
	startTime := time.Now()
	cmd := exec.Command("python", pythonScript) // or use "python3" depending on your environment

	// Run the command and capture the output
	status := db.JobStatusCompleted
	output, err := cmd.CombinedOutput()
	if err != nil {
		status = db.JobStatusFailed
		log.Printf("Error running Python script: %v\nOutput: %s", err, string(output))
	}

	jec, err := controller.NewJobExecutionController("JobExecutionOperationController")
	if err != nil {
		return err
	}

	jobExecution := &db.JobExecution{
		JobID:     jobId,
		WorkerID:  w.Id,
		StartTime: startTime,
		EndTime:   time.Now(),
		Status:    status,
		Error:     err.Error(),
	}
	if err := jec.CreateJobExecution(jobExecution); err != nil {
		return err
	}

	return nil
}


func (wp *WorkerPool) Run() {
	defer close(wp.HighChan)
	defer close(wp.MidChan)
	defer close(wp.LowChan)

	go highBucket.Refill()
	go midBucket.Refill()
	go lowBucket.Refill()

	for i := 0; i < wp.size; i++ {
		wp.wg.Add(1)
		go func() {
            workerId := utils.GenerateWorkerId()
			log.Printf("\nWorker %s created", workerId)
			defer wp.wg.Done()

			for {  
				select {
				case jobId := <-wp.HighChan:
					if highBucket.Take() {
						w := wp.pool.Get().(*Worker)
						w.Id = workerId
						if err := w.Start(jobId); err != nil {
							log.Printf("Error executing high-priority job: %v", err)
						}
						wp.pool.Put(w)
					}
				case jobId := <-wp.MidChan:
					if midBucket.Take() { 
						w := wp.pool.Get().(*Worker)
						w.Id = workerId
						dummyJob(jobId, workerId)
						wp.pool.Put(w)
					}
				case jobId := <-wp.LowChan:
					if lowBucket.Take() {
						w := wp.pool.Get().(*Worker)
						w.Id = workerId
						errorJob(jobId, workerId)
						wp.pool.Put(w)
					}
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	wp.wg.Wait()
}
