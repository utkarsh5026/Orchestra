package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
	"github.com/utkarsh5026/Orchestra/worker"
)

type Manager struct {
	LastWorkerIdx int
	Pending       queue.Queue
	TaskStore     map[uuid.UUID]*task.Task
	EventStore    map[uuid.UUID]*task.Event
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

// NewManager creates and initializes a new Manager instance.
//
// Parameters:
//   - workers: A slice of worker addresses/endpoints that this manager will coordinate
//
// Returns:
//   - *Manager: A new Manager instance initialized with:
func NewManager(workers []string) *Manager {
	taskStore := make(map[uuid.UUID]*task.Task)
	eventStore := make(map[uuid.UUID]*task.Event)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	for _, w := range workers {
		workerTaskMap[w] = []uuid.UUID{}
	}

	return &Manager{
		TaskStore:     taskStore,
		EventStore:    eventStore,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		Workers:       workers,
		Pending:       *queue.New(),
	}
}

// SelectWorker returns the next available worker using round-robin scheduling.
//
// It maintains the LastWorkerIdx to track which worker was last selected and
// cycles through the list of workers sequentially. When it reaches the end,
// it wraps back to the beginning.
//
// Returns:
//   - The address/endpoint of the selected worker
//   - An error if no workers are available
func (m *Manager) SelectWorker() (string, error) {
	if len(m.Workers) == 0 {
		return "", errors.New("no workers available")
	}

	var newWorkerIdx int
	if m.LastWorkerIdx < len(m.Workers) {
		newWorkerIdx = m.LastWorkerIdx + 1
		m.LastWorkerIdx++
	} else {
		newWorkerIdx = 0
		m.LastWorkerIdx = 0
	}
	return m.Workers[newWorkerIdx], nil
}

// UpdateTasks polls all workers for their current tasks and updates the manager's task store
// with any changes to task state or metadata.
//
// For each worker, it:
// 1. Retrieves the current tasks via HTTP GET request
// 2. For each task returned by the worker:
//   - Checks if the task exists in the manager's task store
//   - Updates the task's state and metadata if found
//
// Any errors communicating with workers or tasks not found in the store are logged
// but do not stop processing of other workers/tasks.
func (m *Manager) UpdateTasks() {
	for _, w := range m.Workers {
		log.Printf("Checking worker for task updates: %s", w)
		tasks, err := m.getTasksFromWorker(w)
		if err != nil {
			log.Printf("Error getting tasks from worker %s: %s", w, err)
			continue
		}

		for _, t := range tasks {
			_, ok := m.TaskStore[t.ID]
			if !ok {
				log.Printf("Task %s not found in task store", t.ID)
				continue
			}
			m.updateTask(t)
		}
	}
}

// SendWork dequeues a pending task and sends it to an available worker
//
// Returns:
//   - error if there are no pending tasks, no available workers,
//     task marshaling fails, or sending to worker fails
//
// The function will:
// 1. Check for pending tasks and available workers
// 2. Select a worker using round-robin scheduling
// 3. Dequeue the next task event and update tracking maps
// 4. Set task state to Scheduled
// 5. Marshal and send the task to the selected worker
func (m *Manager) SendWork() error {
	if m.Pending.Len() == 0 {
		return errors.New("no pending tasks")
	}

	worker, err := m.SelectWorker()
	if err != nil {
		return err
	}

	taskEvent := m.Pending.Dequeue().(task.Event)
	t := taskEvent.Task

	log.Printf("Sending task %s to worker %s", t.ID, worker)
	m.EventStore[taskEvent.ID] = &taskEvent
	m.TaskWorkerMap[t.ID] = worker
	m.WorkerTaskMap[worker] = append(m.WorkerTaskMap[worker], t.ID)

	t.State = task.Scheduled
	m.TaskStore[t.ID] = &t

	data, err := json.Marshal(taskEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal task event: %w", err)
	}
	return m.sendTaskToWorker(worker, data)
}

// sendTaskToWorker sends a task to a worker via HTTP POST request
//
// Parameters:
//   - workerName: The name/address of the worker to send the task to
//   - data: JSON encoded task event data to send
//
// Returns:
//   - error if the request fails, the worker returns an error response,
//     or the response cannot be decoded
//
// The function will:
// 1. Send the task data to the worker's /tasks endpoint
// 2. Re-queue the task if the request fails
// 3. Parse and return any error response from the worker
// 4. Decode and log the successful task response
func (m *Manager) sendTaskToWorker(workerName string, data []byte) error {
	url := fmt.Sprintf("http://%s/tasks", workerName)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		m.Pending.Enqueue(data)
		return fmt.Errorf("failed to send task to worker %s: %w", workerName, err)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		var errResp worker.ErrorResponse
		err := decoder.Decode(&errResp)
		if err != nil {
			return fmt.Errorf("failed to decode error response: %w", err)
		}
		return fmt.Errorf("failed to send task to worker %s: %s", workerName, resp.Status)
	}

	var t task.Task
	err = decoder.Decode(&t)
	if err != nil {
		return fmt.Errorf("failed to decode task response: %w", err)
	}

	log.Printf("Task %s sent to worker %s", t.ID, workerName)
	log.Printf("%#v\n", t)
	return nil
}

// updateTask updates the manager's task store with the latest task state and metadata
//
// Parameters:
//   - t: Pointer to the task.Task object to update
//
// The function will:
// 1. Update the task's state and metadata in the manager's task store
func (m *Manager) updateTask(t *task.Task) {
	m.TaskStore[t.ID].StartTime = t.StartTime
	m.TaskStore[t.ID].EndTime = t.EndTime
	m.TaskStore[t.ID].State = t.State
	m.TaskStore[t.ID].ContainerID = t.ContainerID
}

// getTasksFromWorker retrieves the current tasks from a worker via HTTP GET request
//
// Parameters:
//   - workerName: The name/address of the worker to get tasks from
//
// Returns:
//   - []*task.Task: Array of tasks currently running on the worker
//   - error: If the request fails, worker returns non-200 status, or response cannot be decoded
//
// The function will:
// 1. Make an HTTP GET request to the worker's /tasks endpoint
// 2. Check for successful response status
// 3. Decode the JSON response into task objects
func (m *Manager) getTasksFromWorker(workerName string) ([]*task.Task, error) {
	url := fmt.Sprintf("http://%s/tasks", workerName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks from worker %s: %w", workerName, err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting tasks from worker %s: %s", workerName, resp.Status)
	}

	var tasks []*task.Task
	err = json.NewDecoder(resp.Body).Decode(&tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tasks from worker %s: %w", workerName, err)
	}

	return tasks, nil
}

func (m *Manager) AddTask(te task.Event) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := make([]*task.Task, 0, len(m.TaskStore))
	for _, t := range m.TaskStore {
		tasks = append(tasks, t)
	}
	return tasks
}
