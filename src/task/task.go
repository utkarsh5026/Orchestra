package task

import (
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"time"
)

type Task struct {
	ID            uuid.UUID
	State         State
	ContainerID   string
	Name          string
	Image         string
	Memory        int64
	Disk          int64
	Cpu           float64
	ExposedPorts  nat.PortSet
	RestartPolicy string
	PortBindings  map[string]string
	StartTime     time.Time
	EndTime       time.Time
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
	Runtime       Runtime
}

type Runtime struct {
	ContainerId string
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		ExposedPorts:  t.ExposedPorts,
		Image:         t.Image,
		Cpu:           t.Cpu,
		Memory:        t.Memory,
		Disk:          t.Disk,
		RestartPolicy: t.RestartPolicy,
	}
}
