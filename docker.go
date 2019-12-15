package main

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

const (
	dockerAPIVersion = "1.37"
	// dockerDaemonHost = "unix:///var/run/docker.sock"
	dockerDaemonHost = "unix:///host/var/run/docker.sock"
)

// DockerAPI perform various Docker tasks.
type DockerAPI struct {
	cli *client.Client
	ctx context.Context
}

func newDockerAPI() *DockerAPI {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion(dockerAPIVersion), client.WithHost(dockerDaemonHost))
	if err != nil {
		panic(err)
	}

	return &DockerAPI{
		ctx: ctx,
		cli: cli,
	}
}

// RunContainer fire up a new docker container.
func (d *DockerAPI) RunContainer(image string) (string, error) {
	cli := d.cli
	ctx := d.ctx

	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: image,
			Tty:   true,
			// Cmd:   []string{"redis-server", "/usr/local/etc/redis/redis.conf"},
		}, &container.HostConfig{
			Resources: container.Resources{Memory: 1000000 * 10}, // 10 MB memory limit
			Sysctls:   map[string]string{"net.core.somaxconn": "511"},
		},
		nil, "")
	if err != nil {
		return "", err
	}

	containerID := resp.ID

	// redisConfig, err := os.Open("./redis.conf")
	// cli.CopyToContainer(ctx, containerID, "/usr/local/etc/redis/redis.conf", bufio.NewReader(redisConfig), types.CopyToContainerOptions{})

	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return containerID, nil
}

// RemoveContainer removes a container.
func (d *DockerAPI) RemoveContainer(id string) error {
	cli := d.cli
	ctx := d.ctx

	if err := cli.ContainerStop(ctx, id, nil); err != nil {
		return err
	}

	if err := cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	return nil
}

// GetContainerIP retrieve container IP address.
func (d *DockerAPI) GetContainerIP(ID string) (string, error) {
	cli := d.cli
	ctx := d.ctx
	inspect, err := cli.ContainerInspect(ctx, ID)
	if err != nil {
		return "", err
	}
	return inspect.NetworkSettings.IPAddress, nil
}

// GetContainerLogs read container logs.
func (d *DockerAPI) GetContainerLogs(ID string) ([]byte, error) {
	cli := d.cli
	ctx := d.ctx

	reader, err := cli.ContainerLogs(ctx, ID, types.ContainerLogsOptions{ShowStderr: true, ShowStdout: true, Since: "1529235647"})
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	return ioutil.ReadAll(reader)
}
