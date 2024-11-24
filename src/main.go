package main

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/client"
	"github.com/utkarsh5026/Orchestra/task"
)

func createContainer() (*task.Docker, *task.DockerResult) {
	config := task.Config{
		Name:  "test",
		Image: "postgres:15",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
		},
	}

	opts := client.WithAPIVersionNegotiation()
	cl, err := client.NewClientWithOpts(client.FromEnv, opts)

	if err != nil {
		fmt.Printf("Error creating docker client: %v\n\n", err)
		return nil, nil
	}

	d := task.Docker{
		Client: cl,
		Config: config,
	}

	result := d.Run()
	if result.Error != nil {
		fmt.Printf("Error creating container: %v\n\n", result.Error)
		return nil, nil
	}

	fmt.Printf("Container created successfully %s\n\n", result.ContainerId)
	return &d, &result
}

func stopContainer(d *task.Docker, cid string) *task.DockerResult {
	result := d.Stop(cid)
	if result.Error != nil {
		fmt.Printf("Error stopping container %s: %v\n\n", cid, result.Error)
		return nil
	}
	fmt.Printf("Container stopped successfully %s\n\n", cid)
	return &result
}

func main() {
	fmt.Println("Trying to create a container")
	dockerTask, createResult := createContainer()
	if createResult.Error != nil {
		fmt.Printf("%v", createResult.Error)
		os.Exit(1)
	}
	time.Sleep(time.Second * 5)
	fmt.Printf("stopping container %s\n", createResult.ContainerId)
	_ = stopContainer(dockerTask, createResult.ContainerId)
}
