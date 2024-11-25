package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
)

// StartTaskHandler handles HTTP POST requests to create a new task.
//
// It expects a JSON request body containing a task.Event object. The handler will:
// 1. Decode the JSON request body into a task.Event
// 2. Add the task event to the manager's pending queue
// 3. Return the created task with 201 Created status
//
// Returns:
//   - 201 Created with the created task on success
//   - 400 Bad Request if the request body is invalid or malformed
func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	var te task.Event
	if err := d.Decode(&te); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding task event: %v", err), http.StatusBadRequest)
		return
	}

	a.Manager.AddTask(te)
	log.Printf("Task event added: %v", te)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

// GetTasksHandler handles HTTP GET requests to retrieve all tasks.
//
// The handler will:
// 1. Get all tasks from the manager's task store
// 2. Return the tasks as a JSON array with 200 OK status
//
// Returns:
//   - 200 OK with JSON array of all tasks
func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	tasks := a.Manager.GetTasks()
	json.NewEncoder(w).Encode(tasks)
}

// StopTaskHandler handles HTTP DELETE requests to stop a running task.
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request containing the task ID in the URL path
//
// The handler will:
// 1. Extract and validate the task ID from the URL path
// 2. Look up the task in the manager's task store
// 3. Create a new task event with Completed state
// 4. Add the event to the manager's pending queue
//
// Returns:
//   - 204 No Content on successful task stop
//   - 400 Bad Request if task ID is missing or invalid
//   - 404 Not Found if task does not exist
func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid task ID: %v", err), http.StatusBadRequest)
		return
	}

	taskToStop, ok := a.Manager.TaskStore[tID]
	if !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	te := task.Event{
		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy

	a.Manager.AddTask(te)
	log.Printf("Task stopped: %v", te)
	w.WriteHeader(http.StatusNoContent)
}
