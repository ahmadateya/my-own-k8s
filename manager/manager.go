package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ahmadateya/my-own-k8s/task"
	"github.com/ahmadateya/my-own-k8s/worker"
	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type WorkerAddress string // <hostname>:<port>

type Manager struct {
	Pending       queue.Queue               // Pending tasks (stored as task.Event)
	TaskDb        map[uuid.UUID]*task.Task  // [taskID]Task
	EventDb       map[uuid.UUID]*task.Event // [taskID]Event
	Workers       []WorkerAddress
	WorkerTaskMap map[WorkerAddress][]uuid.UUID // [WorkerAddress]taskID
	TaskWorkerMap map[uuid.UUID]WorkerAddress   // [taskID]WorkerAddress
	LastWorker    int
}

func New(workers []WorkerAddress) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.Event)
	workerTaskMap := make(map[WorkerAddress][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]WorkerAddress)
	for w := range workers {
		workerTaskMap[workers[w]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) AddTask(te task.Event) {
	m.Pending.Enqueue(te)
}

func (m *Manager) SelectWorker() WorkerAddress {
	// simple round-robin algorithm
	var newWorker int
	if newWorker+1 < len(m.Workers) { // if there are more workers
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}
	return m.Workers[newWorker]
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking for task updates from workers")
		m.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

// updateTasks => Query the worker to get a list of its tasks, and foreach task update its state in the manager’s DB
func (m *Manager) updateTasks() {
	for _, w := range m.Workers {
		slog.Info(fmt.Sprintf("Checking worker %v for task updates", w))
		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Get(url)
		if err != nil {
			slog.Info(fmt.Sprintf("Error connecting to %v: %v\n", w, err))
		}

		if resp.StatusCode != http.StatusOK {
			slog.Info(fmt.Sprintf("Error sending request: %v\n", err))
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			slog.Info(fmt.Sprintf("Error unmarshalling tasks: %s\n", err.Error()))
		}

		for _, t := range tasks {
			slog.Info(fmt.Sprintf("Attempting to update task %v\n", t.ID))

			_, ok := m.TaskDb[t.ID]
			if !ok {
				slog.Info(fmt.Sprintf("Task with ID %s not found\n", t.ID))
				return
			}
			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}
			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the queue")
		m.SendWork()
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()
		e := m.Pending.Dequeue()
		te := e.(task.Event)
		t := te.Task
		slog.Info(fmt.Sprintf("Pulled %v off pending queue\n", t))

		m.EventDb[te.ID] = &te
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			slog.Error(fmt.Sprintf("Unable to marshal task object: %v.\n", t))
		}
		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			slog.Error(fmt.Sprintf("Error connecting to %v: %v\n", w, err))
			m.Pending.Enqueue(te)
			return
		}
		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := d.Decode(&e)
			if err != nil {
				slog.Error(fmt.Sprintf("Error decoding response: %s\n", err.Error()))
				return
			}
			slog.Info(fmt.Sprintf("Response error (%d): %s", e.HTTPStatusCode, e.Message))
			return
		}

		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			slog.Error("Error decoding response: %s\n", err.Error())
			return
		}
		slog.Info(fmt.Sprintf("%#v\n", t))
	} else {
		slog.Info("No work in the queue")
	}
}

func (m *Manager) checkTaskHealth(t task.Task) error {
	log.Printf("Calling health check for task %s: %s\n", t.ID, t.HealthCheck)

	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)
	workerAddr := strings.Split(string(w), ":")
	if hostPort == nil {
		log.Printf("Have not collected task %s host port yet. Skipping.\n", t.ID)
		return nil
	}
	url := fmt.Sprintf("http://%s:%s%s", workerAddr[0], *hostPort, t.HealthCheck)
	log.Printf("Calling health check for task %s: %s\n", t.ID, url)
	resp, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to health check %s", url)
		log.Println(msg)
		return fmt.Errorf(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Error health check for task %s did not return 200\n", t.ID)
		log.Println(msg)
		return fmt.Errorf(msg)
	}

	log.Printf("Task %s health check response: %v\n", t.ID, resp.StatusCode)

	return nil
}

func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
		return &ports[k][0].HostPort
	}
	return nil
}

func (m *Manager) doHealthChecks() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < 3 {
			err := m.checkTaskHealth(*t)
			if err != nil {
				if t.RestartCount < 3 {
					m.restartTask(t)
				}
			}
		} else if t.State == task.Failed && t.RestartCount < 3 {
			m.restartTask(t)
		}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	// Get the worker where the task was running
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	// We need to overwrite the existing task to ensure it has
	// the current state
	m.TaskDb[t.ID] = t

	te := task.Event{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}
	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Unable to marshal task object: %v.\n", t)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		e := worker.ErrResponse{}
		err := d.Decode(&e)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}
		log.Printf("Response error (%d): %s\n", e.HTTPStatusCode, e.Message)
		return
	}

	newTask := task.Task{}
	err = d.Decode(&newTask)
	if err != nil {
		fmt.Printf("Error decoding response: %s\n", err.Error())
		return
	}
	log.Printf("%#v\n", t)
}

func (m *Manager) DoHealthChecks() {
	for {
		log.Println("Performing task health check")
		m.doHealthChecks()
		log.Println("Task health checks completed")
		log.Println("Sleeping for 60 seconds")
		time.Sleep(60 * time.Second)
	}
}
