package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/hostathome/cli/internal/registry"
)

const containerPrefix = "hostathome-"

// ContainerStatus represents the status of a game container
type ContainerStatus struct {
	Game        string
	Status      string
	Ports       string
	ContainerID string
}

// PullImage pulls the Docker image for a game
func PullImage(imageName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	fmt.Printf("Pulling %s...\n", imageName)
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Consume the output (shows progress)
	io.Copy(os.Stdout, reader)
	return nil
}

// CreateServerDirs creates the directory structure for a game server
func CreateServerDirs(gameName string) error {
	baseDir := fmt.Sprintf("./%s-server", gameName)
	dirs := []string{
		filepath.Join(baseDir, "save"),
		filepath.Join(baseDir, "mods"),
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "configs"),
		filepath.Join(baseDir, "backup"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// RunContainer starts a game server container
func RunContainer(gameName string, game *registry.Game) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	containerName := containerPrefix + gameName

	// Check if container already exists
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return err
	}

	// If container exists, start it
	if len(containers) > 0 {
		c := containers[0]
		if c.State == "running" {
			fmt.Printf("Container %s is already running\n", containerName)
			return nil
		}
		fmt.Printf("Starting existing container %s...\n", containerName)
		return cli.ContainerStart(ctx, c.ID, container.StartOptions{})
	}

	// Create new container
	absPath, err := filepath.Abs(fmt.Sprintf("./%s-server", gameName))
	if err != nil {
		return err
	}

	// Port mappings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	if game.InternalPorts.Player > 0 {
		playerProto := game.Protocols.DefaultProtocol("player")
		internalPort := nat.Port(fmt.Sprintf("%d/%s", game.InternalPorts.Player, playerProto))
		portBindings[internalPort] = []nat.PortBinding{{HostPort: fmt.Sprintf("%d", game.Ports.Player)}}
		exposedPorts[internalPort] = struct{}{}
	}

	if game.InternalPorts.RCON > 0 && game.Ports.RCON > 0 {
		rconProto := game.Protocols.DefaultProtocol("rcon")
		internalPort := nat.Port(fmt.Sprintf("%d/%s", game.InternalPorts.RCON, rconProto))
		portBindings[internalPort] = []nat.PortBinding{{HostPort: fmt.Sprintf("%d", game.Ports.RCON)}}
		exposedPorts[internalPort] = struct{}{}
	}

	config := &container.Config{
		Image:        game.Image,
		ExposedPorts: exposedPorts,
		Labels: map[string]string{
			"hostathome":      "true",
			"hostathome.game": gameName,
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: absPath,
				Target: "/data",
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return err
	}

	return cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
}

// StopContainer stops a game server container
func StopContainer(gameName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	containerName := containerPrefix + gameName

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("container %s not found or not running", containerName)
	}

	return cli.ContainerStop(ctx, containers[0].ID, container.StopOptions{})
}

// GetStatus returns the status of game containers
func GetStatus(gameName string) ([]ContainerStatus, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	filterArgs := filters.NewArgs(filters.Arg("label", "hostathome=true"))
	if gameName != "" {
		filterArgs.Add("name", containerPrefix+gameName)
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	var statuses []ContainerStatus
	for _, c := range containers {
		game := c.Labels["hostathome.game"]
		if game == "" {
			game = strings.TrimPrefix(c.Names[0], "/"+containerPrefix)
		}

		ports := formatPorts(c.Ports)

		statuses = append(statuses, ContainerStatus{
			Game:        game,
			Status:      c.State,
			Ports:       ports,
			ContainerID: c.ID,
		})
	}

	return statuses, nil
}

func formatPorts(ports []types.Port) string {
	var parts []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
