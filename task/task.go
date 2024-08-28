package task

import (
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"time"
)

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID     uuid.UUID
	Name   string
	State  State
	Image  string
	Cpu    float64
	Memory int
	Disk   int
	// ExposedPorts and PortBindings are used by Docker to ensure the machine allocates the proper network ports
	// for the task and that it is available on the network.
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
}

// Event (TaskEvent) an internal object that our system uses to trigger tasks from one state to another.
type Event struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

// Config is a struct that holds the configuration for a task.
type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int // Memory in MiB
	Disk          int // Disk in GiB
	Env           []string
	RestartPolicy string
}
