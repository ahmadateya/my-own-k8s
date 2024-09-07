package worker

import (
	"fmt"
	"github.com/ahmadateya/my-own-k8s/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"log"
	"log/slog"
	"time"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	Stats     *Stats
	TaskCount int
}

func (w *Worker) CollectStats() {
	for {
		slog.Info(fmt.Sprintf("[Worker %s] Collecting stats", w.Name))
		w.Stats = GetStats()
		w.Stats.TaskCount = w.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		slog.Info(fmt.Sprintf("[Worker %s] No tasks in the queue", w.Name))
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	taskPersisted := w.Db[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = fmt.Errorf("we should not get here")
		}
	} else {
		err := fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}
	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks to process currently.\n")
		}
		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		slog.Error("Error creating Docker client: %v\n", err)
		return task.DockerResult{Error: err}
	}
	result := d.Run()
	if result.Error != nil {
		slog.Error("Err running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		slog.Error("Error creating Docker client: %v\n", err)
		return task.DockerResult{Error: err}
	}

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		slog.Error("Error stopping container %v: %v\n", t.ContainerID, result.Error)
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = &t
	slog.Error("Stopped and removed container %v for task %v\n", t.ContainerID, t.ID)
	return result
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range w.Db {
		tasks = append(tasks, t)
	}
	return tasks
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		slog.Error("Error creating Docker client: %v\n", err)
		return task.DockerInspectResponse{Error: err}
	}
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("Checking status of tasks")
		w.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) updateTasks() {
	// for each task in the worker's datastore:
	// 1. call InspectTask method
	// 2. verify task is in running state
	// 3. if task is not in running state, or not running at all, mark task as `failed`
	for id, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				fmt.Printf("ERROR: %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("No container for running task %s\n", id)
				w.Db[id].State = task.Failed
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("Container for task %s in non-running state %s", id, resp.Container.State.Status)
				w.Db[id].State = task.Failed
			}

			w.Db[id].HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
		}
	}
}
