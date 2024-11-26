package worker

import (
	"encoding/json"
	"github.com/utkarsh5026/Orchestra/handler"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
)

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
func (a *Api) StartTaskHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	taskEvent := task.Event{}
	err := d.Decode(&taskEvent)

	if err != nil {
		resErr := handler.Err(http.StatusBadRequest, "Invalid request body", err)
		handler.SendErr(w, resErr)
		return
	}

	a.Worker.AddTask(&taskEvent.Task)
	log.Printf("Task added to the queue: %s", taskEvent.Task.ID)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(taskEvent.Task)
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
func (a *Api) GetTasksHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	ts, err := a.Worker.GetTasks()
	if err != nil {
		resErr := handler.Err(http.StatusInternalServerError, "Error getting tasks", err)
		handler.SendErr(w, resErr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(ts)
	if err != nil {
		log.Printf("Error encoding tasks: %v", err)
		resErr := handler.Err(http.StatusInternalServerError, "Error encoding tasks", err)
		handler.SendErr(w, resErr)
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
func (a *Api) StopTaskHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	taskId := chi.URLParam(r, "taskID")
	if taskId == "" {
		resErr := handler.Err(http.StatusBadRequest, "Task ID is required", nil)
		handler.SendErr(w, resErr)
		return
	}

	tID, err := uuid.Parse(taskId)
	if err != nil {
		resErr := handler.Err(http.StatusBadRequest, "Invalid task ID", err)
		handler.SendErr(w, resErr)
		return
	}

	t, err := a.Worker.Db.Get(tID)
	if err != nil {
		resErr := handler.Err(http.StatusNotFound, "Task not found", err)
		handler.SendErr(w, resErr)
	}

	taskToStop, err := a.Worker.Db.Get(t.ID)
	if err != nil {
		resErr := handler.Err(http.StatusNotFound, "Task not found", err)
		handler.SendErr(w, resErr)
		return
	}

	taskToStop.State = task.Completed
	a.Worker.AddTask(taskToStop)

	log.Printf("Adding task %v to stop the container %v\n", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(http.StatusNoContent)
}
