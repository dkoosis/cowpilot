package rtm

import (
	"testing"
)

func TestEnhancedHandlerCreation(t *testing.T) {
	// Test that enhanced handler can be created
	baseHandler := &Handler{
		client: &Client{
			APIKey: "test",
			Secret: "test",
		},
	}

	eh := NewEnhancedHandler(baseHandler)
	if eh == nil {
		t.Fatal("Failed to create enhanced handler")
	}

	// Only check fields after confirming eh is not nil
	if eh != nil {
		if eh.jobQueue == nil {
			t.Fatal("Job queue not initialized")
		}

		if eh.searchCache == nil {
			t.Fatal("Search cache not initialized")
		}
	}
}

func TestJobQueueCreation(t *testing.T) {
	handler := &Handler{
		client: &Client{
			APIKey: "test",
			Secret: "test",
		},
	}

	queue := NewJobQueue(handler)
	if queue == nil {
		t.Fatal("Failed to create job queue")
	}

	// Test job creation
	job := &BatchJob{
		ID:         "test-123",
		Type:       "batch_due_date",
		Status:     JobStatusPending,
		TotalTasks: 5,
	}

	queue.QueueJob(job)

	retrieved, ok := queue.GetJob("test-123")
	if !ok {
		t.Fatal("Failed to retrieve queued job")
	}

	if retrieved.ID != "test-123" {
		t.Fatalf("Wrong job ID: got %s, want test-123", retrieved.ID)
	}
}
