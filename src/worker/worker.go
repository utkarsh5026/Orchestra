package worker

import (
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/src/task"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) StartTask() {
	// Dequeue a task from the worker's queue
	// and start executing it
}

func (w *Worker) StopTask() {
	// Stop the currently running task
}

func (w *Worker) AddTask(t *task.Task) {
	// Add a task to the worker's queue
}

func (w *Worker) RemoveTask(id uuid.UUID) {
	// Remove a task from the worker's queue
}

func (w *Worker) RunTask() {
	// Run the task
}

func (w *Worker) CollectStats() {
	fmt.Println("I will collect stats")
}
