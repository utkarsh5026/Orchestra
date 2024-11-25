package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/utkarsh5026/Orchestra/store"
	"github.com/utkarsh5026/Orchestra/utils"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        store.Store[uuid.UUID, *task.Task]
	TaskCount int
}

func NewWorker(name string, dt store.Type) *Worker {
	w := Worker{
		Name:  name,
		Queue: *queue.New(),
	}
	w.Db = store.NewStore[uuid.UUID, *task.Task](dt)
	return &w
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
		utils.UpdateStore(w.Db, t.ID, t)
		return task.DockerResult{Error: err}
	}

	result := d.Run()

	if result.Error != nil {
		log.Printf("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		utils.UpdateStore(w.Db, t.ID, t)
		return result
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	utils.UpdateStore(w.Db, t.ID, t)
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
		w.finishTask(t)
		return task.DockerResult{Error: err}
	}

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("Error stopping container %s: %v\n", t.ContainerID, result.Error)
	}

	w.finishTask(t)
	log.Printf("Stopped and removed container %v for task %v\n",
		t.ContainerID, t.ID)
	return result
}

// RunTask processes the next task in the worker's queue.
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
	taskPersisted, err := w.Db.Get(taskToRun.ID)
	if err != nil {
		log.Printf("Error getting task %s: %v\n", taskToRun.ID, err)
		return task.DockerResult{Error: err}
	}

	if taskPersisted == nil {
		taskPersisted = taskToRun
		utils.UpdateStore(w.Db, taskToRun.ID, taskPersisted)
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

// GetTasks returns a slice of all tasks currently stored in the worker's database.
//
// Returns:
//   - []*task.Task: A slice containing pointers to all Task objects in the worker's database
//   - error: Any error that occurred while retrieving tasks from the database
func (w *Worker) GetTasks() ([]*task.Task, error) {
	ts, err := w.Db.List()
	if err != nil {
		return nil, err
	}
	tasks := make([]*task.Task, 0, len(ts))
	for _, t := range ts {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// RunTasks continuously processes tasks from the worker's queue in an infinite loop.
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

// UpdateTasks continuously monitors and updates task status at specified intervals.
//
// Parameters:
//   - d: The duration to wait between status checks
//
// This function runs indefinitely and should be started in a separate goroutine.
func (w *Worker) UpdateTasks(d time.Duration) {
	for {
		log.Println("Checking status of tasks")
		w.updateTasks()
		log.Println("Task updates completed")
		log.Printf("Sleeping for %v seconds\n", d)
		time.Sleep(d)
	}
}

// updateTasks checks all running tasks and updates their state based on container status.
//
// Any errors encountered during listing tasks, inspecting containers, or updating
// task state are logged but do not stop processing of other tasks.
func (w *Worker) updateTasks() {
	tasks, err := w.Db.List()
	if err != nil {
		log.Printf("Error listing tasks: %v\n", err)
		return
	}

	for _, t := range tasks {
		if t.State != task.Running {
			continue
		}

		inspect := w.InspectTask(*t)
		if inspect.Error != nil {
			log.Printf("Error inspecting container %s: %v\n", t.ContainerID, inspect.Error)
			continue
		}

		if inspect.Inspect.State.Status == "exited" {
			log.Printf("Container %s exited with status %d\n", t.ContainerID, inspect.Inspect.State.ExitCode)
			t.State = task.Failed
			utils.UpdateStore(w.Db, t.ID, t)
		}
	}
}

func (w *Worker) finishTask(t *task.Task) error {
	t.State = task.Completed
	t.EndTime = time.Now().UTC()
	return w.Db.Put(t.ID, t)
}
