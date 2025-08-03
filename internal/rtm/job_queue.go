package rtm

import (
	"fmt"
	"sync"
	"time"
)

// JobStatus represents the current state of a batch job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// BatchJob represents a batch operation
type BatchJob struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      JobStatus              `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	TotalTasks  int                    `json:"total_tasks"`
	Completed   int                    `json:"completed"`
	Failed      []string               `json:"failed,omitempty"`
	Results     map[string]interface{} `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// JobQueue manages batch operations
type JobQueue struct {
	mu       sync.RWMutex
	jobs     map[string]*BatchJob
	handler  *Handler
	workers  int
	jobsChan chan string
}

// NewJobQueue creates a new job queue
func NewJobQueue(handler *Handler) *JobQueue {
	q := &JobQueue{
		jobs:     make(map[string]*BatchJob),
		handler:  handler,
		workers:  1, // Single worker to respect RTM rate limits
		jobsChan: make(chan string, 100),
	}

	// Start worker
	go q.worker()

	return q
}

// QueueJob adds a new job to the queue
func (q *JobQueue) QueueJob(job *BatchJob) {
	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()

	// Queue for processing
	q.jobsChan <- job.ID
}

// GetJob retrieves job status
func (q *JobQueue) GetJob(id string) (*BatchJob, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	job, ok := q.jobs[id]
	return job, ok
}

// worker processes jobs from the queue
func (q *JobQueue) worker() {
	for jobID := range q.jobsChan {
		q.processJob(jobID)
	}
}

// processJob executes a batch job
func (q *JobQueue) processJob(jobID string) {
	q.mu.Lock()
	job, ok := q.jobs[jobID]
	if !ok {
		q.mu.Unlock()
		return
	}

	job.Status = JobStatusProcessing
	now := time.Now()
	job.StartedAt = &now
	q.mu.Unlock()

	// Process based on job type
	switch job.Type {
	case "batch_due_date":
		q.processBatchDueDate(job)
	case "batch_priority":
		q.processBatchPriority(job)
	case "batch_complete":
		q.processBatchComplete(job)
	case "batch_tags_add":
		q.processBatchTagsAdd(job)
	case "batch_create":
		q.processBatchCreate(job)
	default:
		q.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = fmt.Sprintf("Unknown job type: %s", job.Type)
		q.mu.Unlock()
	}

	// Mark completion
	q.mu.Lock()
	if job.Status != JobStatusFailed {
		job.Status = JobStatusCompleted
	}
	now = time.Now()
	job.CompletedAt = &now
	q.mu.Unlock()
}

// processBatchDueDate handles batch due date updates
func (q *JobQueue) processBatchDueDate(job *BatchJob) {
	tasks, ok := job.Results["tasks"].([]map[string]string)
	if !ok {
		q.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = "Invalid or missing tasks data"
		q.mu.Unlock()
		return
	}

	dueDate, ok := job.Results["due_date"].(string)
	if !ok {
		q.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = "Invalid or missing due_date"
		q.mu.Unlock()
		return
	}

	for i, task := range tasks {
		// Update job progress
		q.mu.Lock()
		job.Completed = i
		q.mu.Unlock()

		// RTM rate limit
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		// Update task
		updates := map[string]string{"due": dueDate}
		err := q.handler.client.UpdateTask(task["list_id"], task["series_id"], task["task_id"], updates)
		if err != nil {
			q.mu.Lock()
			job.Failed = append(job.Failed, fmt.Sprintf("Task %s: %v", task["task_id"], err))
			q.mu.Unlock()
		}
	}

	q.mu.Lock()
	job.Completed = len(tasks)
	q.mu.Unlock()
}

// Similar implementations for other batch operations...
func (q *JobQueue) processBatchPriority(job *BatchJob) {
	// Implementation similar to processBatchDueDate
}

func (q *JobQueue) processBatchComplete(job *BatchJob) {
	// Implementation similar to processBatchDueDate
}

func (q *JobQueue) processBatchTagsAdd(job *BatchJob) {
	// Implementation similar to processBatchDueDate
}

func (q *JobQueue) processBatchCreate(job *BatchJob) {
	taskTexts, ok := job.Results["tasks"].([]string)
	if !ok {
		q.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = "Invalid or missing tasks data"
		q.mu.Unlock()
		return
	}

	for i, taskText := range taskTexts {
		// Update progress
		q.mu.Lock()
		job.Completed = i
		q.mu.Unlock()

		// RTM rate limit
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		// Create task
		_, err := q.handler.client.AddTask(taskText, "")
		if err != nil {
			q.mu.Lock()
			job.Failed = append(job.Failed, fmt.Sprintf("Task '%s': %v", taskText, err))
			q.mu.Unlock()
		}
	}

	q.mu.Lock()
	job.Completed = len(taskTexts)
	q.mu.Unlock()
}
