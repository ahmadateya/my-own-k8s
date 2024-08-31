package task

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
)

const MaxRestartPolicy = 5

var RestartPolicyMap = map[string]container.RestartPolicyMode{
	"":               container.RestartPolicyDisabled,
	"no":             container.RestartPolicyDisabled,
	"always":         container.RestartPolicyAlways,
	"unless-stopped": container.RestartPolicyUnlessStopped,
	"on-failure":     container.RestartPolicyOnFailure,
}

// Docker is a struct that encapsulates everything we need to run our task as a Docker container
type Docker struct {
	Client *client.Client
	Config Config
}

func NewDocker(c *Config) (*Docker, error) {
	dc, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		slog.Error("Error creating Docker client: %v\n", err)
		return nil, err
	}
	return &Docker{
		Client: dc,
		Config: *c,
	}, err
}

// DockerResult is a wrapper around the common information that is useful for callers
type DockerResult struct {
	Error       error
	Action      string // used to identify the action being taken, for example, start or stop
	ContainerId string
	Result      string
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(
		ctx, d.Config.Image, image.PullOptions{})
	if err != nil {
		slog.Error("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	io.Copy(os.Stdout, reader)

	maximumRetryCount := 0
	if d.Config.RestartPolicy == string(container.RestartPolicyOnFailure) {
		maximumRetryCount = MaxRestartPolicy
	}
	rp := container.RestartPolicy{
		Name:              RestartPolicyMap[d.Config.RestartPolicy],
		MaximumRetryCount: maximumRetryCount,
	}

	r := container.Resources{
		Memory:   int64(d.Config.Memory),
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}

	cc := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
	}

	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		slog.Error("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		slog.Error("Error starting container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	out, err := d.Client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true, ShowStderr: true,
		Timestamps: true, Details: true,
	})
	if err != nil {
		slog.Error("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return DockerResult{ContainerId: resp.ID, Action: "start", Result: "success"}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop container %v", id)
	ctx := context.Background()
	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		slog.Error("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	})
	if err != nil {
		slog.Error("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	return DockerResult{Action: "stop", Result: "success", Error: nil}
}
