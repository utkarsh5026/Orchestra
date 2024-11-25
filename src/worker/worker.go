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

func (w *Worker) AddTask(t *task.Task) {
	w.Queue.Enqueue(t)
}

// RunTask dequeues and processes the next task from the worker's queue
//
// Returns:
//   - task.DockerResult containing any errors that occurred during task execution
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

func finishTask(t *task.Task, w *Worker) {
	// Mark the task as completed
	t.State = task.Completed
	t.EndTime = time.Now().UTC()
	w.Db[t.ID] = t
}
