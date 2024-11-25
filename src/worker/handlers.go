package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
)

type ErrorResponse struct {
	HTTPStatusCode int
	Message        string
}

// StartTaskHandler handles HTTP POST requests to start a new task
// It decodes the task event from the request body and adds the task to the worker's queue
//
// Parameters:
//   - w: HTTP response writer to send the response
//   - r: HTTP request containing the task event in its body
//
// The handler expects a JSON request body containing a task.Event
// Returns HTTP 400 if request body is invalid
// Returns HTTP 200 with the queued task on success
func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	taskEvent := task.Event{}
	err := d.Decode(&taskEvent)

	if err != nil {
		msg := fmt.Sprintf("Error decoding the request body: %v", err)
		log.Println(msg)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        msg,
		})
		return
	}

	a.Worker.AddTask(&taskEvent.Task)
	log.Printf("Task added to the queue: %s", taskEvent.Task.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(taskEvent.Task)
}

// GetTasksHandler handles HTTP GET requests to retrieve all tasks from the worker
// It returns a JSON array of all tasks currently tracked by the worker
//
// Parameters:
//   - w: HTTP response writer to send the response
//   - r: HTTP request (unused)
//
// Returns HTTP 200 with JSON array of tasks on success
// Returns HTTP 500 if there is an error encoding the response
func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(a.Worker.GetTasks())
	if err != nil {
		log.Printf("Error encoding tasks: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			HTTPStatusCode: http.StatusInternalServerError,
			Message:        "Error encoding tasks",
		})
	}
}

// StopTaskHandler handles HTTP DELETE requests to stop a running task
// It retrieves the task ID from the URL parameters, validates it, and queues the task for stopping
//
// Parameters:
//   - w: HTTP response writer to send the response
//   - r: HTTP request containing the task ID in the URL path
//
// The handler expects a valid UUID as the taskID URL parameter
// Returns HTTP 400 if task ID is missing or invalid
// Returns HTTP 404 if task is not found
// Returns HTTP 204 on successful queueing of the stop request
func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskID")
	if taskId == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "Task ID is required",
		})
		return
	}

	tID, err := uuid.Parse(taskId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "Invalid task ID",
		})
	}

	_, ok := a.Worker.Db[tID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			HTTPStatusCode: http.StatusNotFound,
			Message:        "Task not found",
		})
	}

	taskToStop := *a.Worker.Db[tID]
	taskToStop.State = task.Completed
	a.Worker.AddTask(&taskToStop)

	log.Printf("Adding task %v to stop the container %v\n", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(http.StatusNoContent)
}
