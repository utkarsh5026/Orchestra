package task

import (
	"context"
	"github.com/docker/docker/api/types"
	"io"
	"log"
	"math"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Docker struct {
	Config Config
	Client *client.Client
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}

type DockerInspectResponse struct {
	Error   error
	Inspect types.ContainerJSON
}

func NewDocker(config Config) (*Docker, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker c: %v\n", err)
		return nil, err
	}
	return &Docker{Config: config, Client: c}, nil
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	img := d.Config.Image
	reader, err := d.Client.ImagePull(ctx,
		img, image.PullOptions{})

	if err != nil {
		log.Printf("Error pulling image %s: %v\n", img, err)
		return DockerResult{Error: err}
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Printf("Error copying image pull response: %v\n", err)
		return DockerResult{Error: err}
	}

	resPo := container.RestartPolicy{
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}

	resource := container.Resources{
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}

	cc := container.Config{
		Image:        d.Config.Image,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
		Cmd:          d.Config.Cmd,
		Tty:          false,
	}

	hc := container.HostConfig{
		RestartPolicy:   resPo,
		Resources:       resource,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf("Error creating container: %v\n", err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		log.Printf("Error starting container: %v\n", err)
		return DockerResult{Error: err}
	}

	d.Config.Runtime.ContainerId = resp.ID
	out, err := d.Client.ContainerLogs(ctx, resp.ID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true})

	if err != nil {
		log.Printf("Error getting container logs: %v\n", err)
		return DockerResult{Error: err}
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		log.Printf("Error copying container logs: %v\n", err)
		return DockerResult{Error: err}
	}

	return DockerResult{ContainerId: resp.ID,
		Action: "start",
		Result: "success"}
}

func (d *Docker) Stop(cid string) DockerResult {
	log.Printf("Stopping container %s\n", cid)
	ctx := context.Background()
	err := d.Client.ContainerStop(ctx, cid, container.StopOptions{})

	if err != nil {
		log.Printf("Error stopping container: %v\n", err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerRemove(ctx, cid, container.RemoveOptions{
		Force:         false,
		RemoveLinks:   true,
		RemoveVolumes: true,
	})

	if err != nil {
		log.Printf("Error removing container: %v\n", err)
		return DockerResult{Error: err}
	}

	return DockerResult{ContainerId: cid,
		Action: "stop",
		Result: "success"}
}

func (d *Docker) Inspect(cid string) DockerInspectResponse {
	ctx := context.Background()
	inspect, err := d.Client.ContainerInspect(ctx, cid)
	if err != nil {
		log.Printf("Error inspecting container %s: %v\n", cid, err)
		return DockerInspectResponse{Error: err}
	}
	return DockerInspectResponse{Inspect: inspect}
}
