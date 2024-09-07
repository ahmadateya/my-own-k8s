package task

import (
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"time"
)

type Task struct {
	ID     uuid.UUID
	Name   string
	State  State
	Image  string
	Cpu    float64
	Memory uint64
	Disk   uint64
	// ExposedPorts and PortBindings are used by Docker to ensure the machine allocates the proper network ports
	// for the task and that it is available on the network.
	ExposedPorts  nat.PortSet
	HostPorts     nat.PortMap
	PortBindings  map[string]string
	RestartPolicy string
	ContainerID   string
	StartTime     time.Time
	FinishTime    time.Time
	HealthCheck   string
	RestartCount  int
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
	Memory        uint64 // Memory in MiB
	Disk          uint64 // Disk in GiB
	Env           []string
	RestartPolicy string
}

// NewConfig creates a new Config object from a Task object.
func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		AttachStdin:   false,
		AttachStdout:  true,
		AttachStderr:  true,
		ExposedPorts:  t.ExposedPorts,
		Cmd:           []string{},
		Image:         t.Image,
		Cpu:           t.Cpu,
		Memory:        t.Memory,
		Disk:          t.Disk,
		Env:           []string{},
		RestartPolicy: t.RestartPolicy,
	}
}
