package task

import (
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"time"
)

type State uint

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID            uuid.UUID
	State         State
	Name          string
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	RestartPolicy string
	PortBindings  map[string]string
	StartTime     time.Time
	EndTime       time.Time
}
