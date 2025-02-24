package scheduler

/*

1. Architecture Overview
The system consists of the following components:

Job Store: A distributed database to store cron job definitions, metadata, and execution history.

Scheduler: A distributed component responsible for triggering jobs based on their schedules and priorities.

Worker Nodes: Nodes that execute the jobs.

Priority Manager: A component that assigns and manages job priorities dynamically.

Distributed Locking Service: Ensures exclusive access to shared resources (e.g., job execution).

Monitoring and Logging: Tracks job execution, system health, and performance metrics.

Load Balancer: Distributes jobs across worker nodes based on priority and load.

2. Job Store
Design Choice: Use a distributed database like Apache Cassandra or CockroachDB.

Scalability: Distributed databases can handle large volumes of job definitions and metadata.

High Availability: Replication ensures data is available even if some nodes fail.

Consistency: Strong consistency guarantees ensure that job definitions are accurate and up-to-date.

Durability: Jobs are persisted and can be recovered in case of failures.

3. Scheduler
Design Choice: Use a distributed scheduling algorithm with a leader-follower model.

Leader Election: Use a consensus algorithm like Raft or Zab to elect a leader responsible for scheduling decisions.

Fault Tolerance: If the leader fails, a new leader is elected automatically.

Efficient Scheduling: The leader maintains a priority queue of jobs and schedules them based on their priority and execution time.

Time Synchronization: Use NTP or PTP to ensure all nodes have synchronized clocks for accurate scheduling.

4. Worker Nodes
Design Choice: Use a container orchestration platform like Kubernetes or Nomad.

Scalability: Easily scale worker nodes up or down based on load.

Isolation: Each job runs in its own container, ensuring isolation and security.

Resource Management: Kubernetes can enforce resource limits and constraints for each job.

Automatic Recovery: Failed jobs can be rescheduled automatically.

5. Priority Manager
Design Choice: Implement a dynamic priority scoring system.

Priority Factors: Consider factors like job type, SLA, deadline, and resource requirements.

Weighted Fair Queuing: Ensures high-priority jobs are executed first while still allowing lower-priority jobs to make progress.

Dynamic Adjustment: Priorities can be adjusted at runtime based on system load or business rules.

6. Distributed Locking Service
Design Choice: Use etcd or Zookeeper for distributed locking.

Exclusive Access: Ensures that only one worker node executes a job at a time.

Fault Tolerance: Distributed locking services are highly available and fault-tolerant.

Consistency: Provides strong consistency guarantees for locking.

7. Monitoring and Logging
Design Choice: Use Prometheus for monitoring and ELK Stack (Elasticsearch, Logstash, Kibana) for logging.

Real-time Monitoring: Track job execution, system health, and performance metrics.

Alerting: Set up alerts for failed jobs, high latency, or system failures.

Centralized Logging: Aggregate logs from all components for easier debugging and analysis.

8. Load Balancer
Design Choice: Use a distributed load balancer like HAProxy or Envoy.

Scalability: Can handle a large number of job submissions and executions.

Health Checks: Monitors worker node health and routes jobs only to healthy nodes.

Priority-Based Routing: Ensures high-priority jobs are routed to the most available and capable worker nodes.

9. Communication Protocol
Design Choice: Use gRPC for inter-component communication.

Efficiency: gRPC is highly efficient and supports bidirectional streaming.

Interoperability: Works well with modern distributed systems.

Security: Supports TLS for secure communication.

10. Fault Tolerance and Recovery
Design Choice: Implement redundancy, checkpointing, and automatic failover.

Redundancy: Run multiple instances of each component to ensure high availability.

Checkpointing: Periodically save the state of jobs to allow for recovery in case of failure.

Automatic Failover: Use leader election and consensus algorithms to automatically failover to a backup instance if the primary fails.

11. Security
Design Choice: Implement RBAC (Role-Based Access Control) and encryption.

RBAC: Ensures that only authorized users and systems can submit or manage jobs.

Encryption: Encrypt data in transit and at rest to protect sensitive information.

Audit Logging: Log all actions for auditing and compliance purposes.

12. Scalability Considerations
Horizontal Scaling: Add more worker nodes and partitions for the job store as the load increases.

Elastic Scaling: Automatically scale the number of worker nodes based on the current load.

Partitioning: Partition the job store and scheduler to distribute the load across multiple nodes.

13. Performance Optimization
Caching: Cache frequently accessed job metadata to reduce latency.

Batch Processing: Process jobs in batches to reduce overhead.

Asynchronous Processing: Use asynchronous communication to avoid blocking.

14. Testing and Validation
Load Testing: Simulate high load to ensure the system can handle it.

Failure Testing: Simulate node failures to ensure the system can recover.

Integration Testing: Ensure all components work together as expected.

15. Example Workflow
Job Submission: A user submits a cron job with a schedule and priority.

Job Store: The job is stored in the distributed database.

Scheduler: The leader scheduler picks up the job, assigns it a priority, and adds it to the priority queue.

Worker Node: The scheduler assigns the job to a worker node based on priority and load.

Execution: The worker node executes the job and updates the job store with the result.

Monitoring: The monitoring system tracks job execution and alerts if any issues arise.

Job Scheduler: The core component responsible for managing job schedules, execution, and monitoring.
Job Queue: A queue where jobs are placed for execution. This queue can be prioritized based on job priority or cron expression.
Event System: A messaging system that handles events such as job creation, job update, job cancellation, and job execution.
Worker: A service that processes and executes jobs based on the schedule.
Cron Expression Evaluator: A component that updates the next run time for jobs when they are created, updated, or modified.
Event Bus: A message broker (e.g., Kafka, RabbitMQ, or Redis Pub/Sub) that facilitates communication between components by publishing and subscribing to events.

3.1 Job Creation Flow
Job Created Event:
The job is created via an API request or other user interaction.
The event system publishes a JobCreated event.
The event listener on the Job Scheduler receives the event and evaluates the cron expression to calculate the next run time.
The Job Scheduler updates the job's next execution time in the database.
The Event System publishes a JobScheduled event with the next run time.
The Worker receives the JobScheduled event and schedules the job for execution.
3.2 Job Update Flow
Job Updated Event:
A job is updated (e.g., its cron expression is modified).
The system publishes a JobUpdated event.
The Cron Expression Evaluator listens to the event and re-calculates the next execution time based on the new cron expression.
The Job Scheduler updates the job with the new next run time and triggers the JobScheduled event for re-scheduling.
The Event System also updates the queue with the new schedule.
3.3 Job Execution Flow
Job Execution Event:
A job is picked up by the Worker, executed, and the system listens for the job completion event (success or failure).
Upon job completion, a JobExecuted event is triggered, indicating the result of the execution.
The Job Scheduler processes the result and checks whether the job needs to be re-scheduled based on its cron expression (for repeated jobs).
If the job is a recurring job, the Cron Expression Evaluator calculates the next scheduled run time.
A new JobScheduled event is published with the updated next execution time.
3.4 Job Cancellation Flow
Job Canceled Event:
If a job is canceled, the system publishes a JobCanceled event.
The Job Scheduler listens to the event, removes the job from the job queue, and updates the database to reflect the job’s canceled status.
The job is no longer scheduled for execution.

1. Fetch all jobs in the db that is due for running
2. Sort by priority
3. Divide by number of priority level queues
4. Distributed jobs to priority queues

*/

import (
	"doit/internal/db"
	"doit/internal/controller"
	"doit/pkg/utils"
	"log"
	"time"
)

type Scheduler struct {
	jobChan chan *db.Job
}

func NewScheduler() *Scheduler {
	return &Scheduler{jobChan: make(chan *db.Job)}
}

func (s *Scheduler) Run() {
	go func() {
		limit := 0
		for {
			jobs, err := db.GetAllJobs(limit, 10)
			if err != nil {
				log.Printf("Error fetching jobs from database: %v", err)
			}
			for _, job := range jobs {
				s.jobChan <- &job
			}
			time.Sleep(3 * time.Second)
			limit += 10
		}
	}()

	for job := range s.jobChan {
        if job.TriggerAt.Compare(time.Now()) == 1 || job.FinishAt.Compare(time.Now()) != 1 {
            continue
        }
		nextRunTime, err := utils.EvalCronExpr(job.CronExpr)
		if err != nil {
			log.Printf("Error evaluating cron for job %s: %v", job.JobID, err)
			continue
		}
		sc, err := controller.CreateScheduleController("ScheduleOperationController")
		if err != nil {
			log.Fatalf("error initializing ScheduleOperationController")
		}
		if err := sc.CreateSchedule(&db.Schedule{
			JobID:       job.JobID,
			Priority:    job.Priority,
			Payload:     job.Payload,
			MaxRetries:  job.MaxRetries,
			RcreTime:    job.RcreTime,
			NextRunTime: nextRunTime,
		}); err != nil {
			log.Printf("Error creating schedule for job %s: %v", job.JobID, err)
		}
	}
}
