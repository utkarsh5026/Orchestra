package manager

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
)

type Manager struct {
	Pending       queue.Queue
	TaskStore     map[string][]*task.Task
	EventStore    map[string][]*task.Event
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

func (m *Manager) SelectWorker() {
	fmt.Println("I will select an appropriate worker")
}
func (m *Manager) UpdateTasks() {
	fmt.Println("I will update tasks")
}
func (m *Manager) SendWork() {
	fmt.Println("I will send work to workers")
}
