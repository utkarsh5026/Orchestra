package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

// StartTask initializes and runs a new task in a Docker container
// Parameters:
//   - t: The task.Task to be started and executed
//
// Returns:
//   - task.DockerResult containing the container ID and any errors that occurred during startup
func (w *Worker) StartTask(t *task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(t)
	d, err := task.NewDocker(*config)

	if err != nil {
		log.Printf("Error creating Docker: %v\n", err)
		t.State = task.Failed
		w.Db[t.ID] = t
		return task.DockerResult{Error: err}
	}

	result := d.Run()

	if result.Error != nil {
		log.Printf("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = t
		return result
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = t
	return result
}

// StopTask stops and removes a running Docker container for a task
// Parameters:
//   - t: The task whose container should be stopped
//
// Returns:
//   - task.DockerResult containing the container ID and any errors that occurred during shutdown
func (w *Worker) StopTask(t *task.Task) task.DockerResult {
	config := task.NewConfig(t)
	d, err := task.NewDocker(*config)
	if err != nil {
		log.Printf("Error creating Docker: %v\n", err)
		finishTask(t, w)
		return task.DockerResult{Error: err}
	}

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("Error stopping container %s: %v\n", t.ContainerID, result.Error)
	}

	finishTask(t, w)
	log.Printf("Stopped and removed container %v for task %v\n",
		t.ContainerID, t.ID)
	return result
}

// RunTask processes the next task in the worker's queue.
//
// The function will:
// 1. Dequeue the next task from the worker's queue
// 2. Check if the task exists in the worker's database
// 3. Validate and execute the requested state transition
//
// State transitions:
//   - Scheduled -> Running: Starts the task's container via StartTask()
//   - Running -> Completed: Stops the task's container via StopTask()
//
// Returns:
//   - task.DockerResult containing:
//   - ContainerId of the started/stopped container
//   - Error if:
//   - Queue is empty
//   - Invalid state transition requested
//   - Docker operations fail
func (w *Worker) RunTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("No tasks to run right now")
		return task.DockerResult{Error: nil}
	}

	taskToRun := t.(*task.Task)
	taskPersisted := w.Db[taskToRun.ID]

	if taskPersisted == nil {
		taskPersisted = taskToRun
		w.Db[taskToRun.ID] = taskPersisted
	}

	var result task.DockerResult

	if taskPersisted.State.CanTransitionTo(taskToRun.State) {
		switch taskToRun.State {
		case task.Scheduled:
			result = w.StartTask(taskToRun)
		case task.Completed:
			result = w.StopTask(taskToRun)
		default:
			err := fmt.Errorf("invalid state transition: %v -> %v", taskPersisted.State, taskToRun.State)
			result.Error = err
		}
	} else {
		err := fmt.Errorf("invalid state transition: %v -> %v", taskPersisted.State, taskToRun.State)
		result.Error = err
	}

	return result
}

func (w *Worker) CollectStats() {
	fmt.Println("I will collect stats")
}

// GetTasks returns a slice of all tasks currently stored in the worker's database.
//
// Returns:
//   - []*task.Task: A slice containing pointers to all Task objects in the worker's database
func (w *Worker) GetTasks() []*task.Task {
	tasks := make([]*task.Task, 0, len(w.Db))
	for _, t := range w.Db {
		tasks = append(tasks, t)
	}
	return tasks
}

// RunTasks continuously processes tasks from the worker's queue in an infinite loop.
//
// The function will:
// 1. Check if there are any tasks in the queue
// 2. If tasks exist, run them one by one
// 3. Sleep for 10 seconds between iterations
//
// This function runs indefinitely and should be started in a separate goroutine.
// It provides the main task processing loop for the worker.
func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() > 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Println("No tasks to process currently.")
		}

		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}

// InspectTask inspects a Docker container associated with a task.
//
// Parameters:
//   - t: The task.Task object containing the container ID to inspect
//
// Returns:
//   - task.DockerInspectResponse containing container inspection details or error
func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d, err := task.NewDocker(*config)
	if err != nil {
		log.Printf("Error creating Docker: %v\n", err)
		return task.DockerInspectResponse{Error: err}
	}
	return d.Inspect(t.ContainerID)
}

func (w *Worker) AddTask(t *task.Task) {
	w.Queue.Enqueue(t)
}

func finishTask(t *task.Task, w *Worker) {
	// Mark the task as completed
	t.State = task.Completed
	t.EndTime = time.Now().UTC()
	w.Db[t.ID] = t
}
