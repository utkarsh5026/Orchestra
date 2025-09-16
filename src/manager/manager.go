package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/utkarsh5026/Orchestra/store"

	"github.com/utkarsh5026/Orchestra/node"
	"github.com/utkarsh5026/Orchestra/scheduler"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/utkarsh5026/Orchestra/task"
	"github.com/utkarsh5026/Orchestra/worker"
)

type Manager struct {
	LastWorkerIdx int
	Pending       queue.Queue
	TaskStore     store.Store[string, *task.Task]
	EventStore    store.Store[string, *task.Event]
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	Scheduler     scheduler.Scheduler
	WorkerNodes   []*node.Node
}

// NewManager creates and initializes a new Manager instance.
//
// Parameters:
//   - workers: A slice of worker addresses/endpoints that this manager will coordinate
//   - st: The type of scheduler to use
//   - storeType: The type of store to use for task and event data
//
// Returns:
//   - *Manager: A new Manager instance initialized with:
func NewManager(workers []string, st scheduler.Type, storeType store.Type) *Manager {
	ts := store.NewStore[string, *task.Task](storeType)
	es := store.NewStore[string, *task.Event](storeType)
	wt := make(map[string][]uuid.UUID)
	tw := make(map[uuid.UUID]string)

	var workerNodes []*node.Node
	for _, w := range workers {
		wt[w] = []uuid.UUID{}
		api := fmt.Sprintf("http://%s/tasks", w)
		n := node.NewNode(w, api, "worker")
		workerNodes = append(workerNodes, n)
	}

	return &Manager{
		TaskStore:     ts,
		EventStore:    es,
		WorkerTaskMap: wt,
		TaskWorkerMap: tw,
		Workers:       workers,
		Pending:       *queue.New(),
		WorkerNodes:   workerNodes,
		Scheduler:     scheduler.NewScheduler(st),
	}
}

// SelectWorker returns the next available worker using round-robin scheduling.
//
// Parameters:
//   - t: The task to select a worker for
//
// Returns:
//   - *node.Node: The selected worker node
//   - An error if no workers are available
func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidates(t, m.WorkerNodes)
	if candidates == nil {
		return nil, fmt.Errorf("No candidates found to satisfy task requirements for the task %v\n", t.ID)
	}

	scores := m.Scheduler.Score(t, candidates)
	if scores == nil {
		return nil, fmt.Errorf("No scores found for the task %v\n", t.ID)
	}

	selected := m.Scheduler.Pick(scores, candidates)
	return selected, nil
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
			old, err := m.TaskStore.Get(t.ID.String())
			if err != nil {
				log.Printf("Task %s not found in task store", t.ID)
				continue
			}
			if err := m.updateTask(old, t); err != nil {
				log.Printf("Error updating task %s: %s", t.ID, err)
			}
		}
	}
}

// SendWork dequeues a pending task and sends it to an available worker
//
// Returns:
//   - error if there are no pending tasks, no available workers,
//     task marshaling fails, or sending to worker fails
func (m *Manager) SendWork() error {
	if m.Pending.Len() == 0 {
		return errors.New("no pending tasks")
	}

	e := m.Pending.Dequeue().(task.Event)
	err := m.EventStore.Put(e.ID.String(), &e)
	if err != nil {
		return fmt.Errorf("failed to persist task event: %w", err)
	}
	log.Printf("Sending task %s to worker\n", e.Task.ID)

	taskID := e.Task.ID
	taskWorker, ok := m.TaskWorkerMap[taskID]
	if ok {
		pt, err := m.TaskStore.Get(taskID.String())
		if err != nil {
			return fmt.Errorf("failed to get persisted task %s: %w", taskID, err)
		}

		if e.State == task.Completed && pt.State.CanTransitionTo(e.State) {
			return m.stopTask(taskWorker, taskID.String())
		}
		return fmt.Errorf("invalid request: existing task %s is in state %v and cannot transition to the completed state", pt.ID.String(), pt.State)
	}

	w, err := m.SelectWorker(e.Task)
	if err != nil {
		return fmt.Errorf("failed to select worker for task %s: %w", taskID, err)
	}

	taskEvent := m.Pending.Dequeue().(task.Event)
	t := taskEvent.Task
	workerName := w.Name
	m.TaskWorkerMap[t.ID] = workerName
	m.WorkerTaskMap[workerName] = append(m.WorkerTaskMap[workerName], t.ID)

	t.State = task.Scheduled
	m.TaskStore.Put(t.ID.String(), &t)

	data, err := json.Marshal(taskEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal task event: %w", err)
	}
	return m.sendTaskToWorker(workerName, data)
}

// updateTask updates the manager's task store with the latest task state and metadata
//
// Parameters:
//   - old: Pointer to the task.Task object to update
//   - new: Pointer to the task.Task object with the updated state and metadata
//
// Returns:
//   - error if the task store update fails
func (m *Manager) updateTask(old *task.Task, new *task.Task) error {
	if old.State != new.State {
		old.State = new.State
	}
	old.StartTime = new.StartTime
	old.EndTime = new.EndTime
	old.State = new.State
	old.ContainerID = new.ContainerID
	return m.TaskStore.Put(old.ID.String(), old)
}

// getTasksFromWorker retrieves the current tasks from a worker via HTTP GET request
//
// Parameters:
//   - workerName: The name/address of the worker to get tasks from
//
// Returns:
//   - []*task.Task: Array of tasks currently running on the worker
//   - error: If the request fails, worker returns non-200 status, or response cannot be decoded
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

// GetTasks returns a slice of all tasks currently stored in the manager's task store.
//
// Returns:
//   - []*task.Task: A slice containing pointers to all Task objects in the manager's task store
func (m *Manager) GetTasks() ([]*task.Task, error) {
	n, err := m.TaskStore.Count()
	if err != nil {
		return nil, err
	}

	ts, err := m.TaskStore.List()
	if err != nil {
		return nil, err
	}
	tasks := make([]*task.Task, 0, n)
	for _, t := range ts {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (m *Manager) LoopTasks() {
	for {
		log.Println("Processing any tasks in the queue")
		err := m.SendWork()

		if err != nil {
			err = fmt.Errorf("error processing tasks: %w", err)
			log.Println(err)
		}

		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

// stopTask sends a request to stop a specific task on a worker node
//
// Parameters:
//   - workerName: The name/address of the worker running the task
//   - taskID: The ID of the task to stop
//
// Returns:
//   - error: If the request fails, worker returns non-204 status, or other errors occur
func (m *Manager) stopTask(workerName string, taskID string) error {
	var httpClient http.Client
	url := fmt.Sprintf("http://%s/tasks/%s", workerName, taskID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to stop task %s on worker %s: %w", taskID, workerName, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop task %s on worker %s: %w", taskID, workerName, err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to stop task %s on worker %s: %s", taskID, workerName, resp.Status)
	}

	log.Printf("Task %s stopped on worker %s", taskID, workerName)
	return nil
}

// restartTask attempts to restart a task on its assigned worker
//
// Parameters:
//   - t: The task to restart
//
// Returns:
//   - error: If the task is not found in the worker map, task state update fails,
//     event marshaling fails, or sending to worker fails
func (m *Manager) restartTask(t *task.Task) error {
	w, ok := m.TaskWorkerMap[t.ID]
	if !ok {
		return fmt.Errorf("task %s not found", t.ID)
	}

	t.State = task.Scheduled
	err := m.TaskStore.Put(t.ID.String(), t)
	if err != nil {
		return fmt.Errorf("failed to update task %s: %w", t.ID, err)
	}

	te := task.Event{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}
	data, err := json.Marshal(te)
	if err != nil {
		return fmt.Errorf("failed to marshal task event: %w", err)
	}
	return m.sendTaskToWorker(w, data)
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
	return nil
}
