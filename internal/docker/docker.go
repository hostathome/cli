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

	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Silently consume the output (spinner shows progress instead)
	_, err = io.Copy(io.Discard, reader)
	return err
}

// CreateServerDirs creates the directory structure for a game server
func CreateServerDirs(gameName string) error {
	baseDir := fmt.Sprintf("./%s-server", gameName)
	dirs := []string{
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "configs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// RunContainer starts a game server container
func RunContainer(gameName string, game *registry.Game, devMode bool) error {
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

	// In dev mode, skip image pull and verify local image exists
	if devMode {
		fmt.Println("🔧 Dev mode: skipping image pull, using local image")
		images, err := cli.ImageList(ctx, image.ListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", game.Image)),
		})
		if err != nil || len(images) == 0 {
			return fmt.Errorf("local image %s not found. Build it first with: docker build -t %s .", game.Image, game.Image)
		}
	} else {
		// Normal mode: pull the image from registry
		if err := PullImage(game.Image); err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
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
				Source: filepath.Join(absPath, "data"),
				Target: "/data",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(absPath, "configs"),
				Target: "/configs",
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

// RestartContainer restarts a game server container
func RestartContainer(gameName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	containerName := containerPrefix + gameName

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("container %s not found", containerName)
	}

	return cli.ContainerRestart(ctx, containers[0].ID, container.StopOptions{})
}

// RemoveContainer removes a game server container but keeps the data
func RemoveContainer(gameName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	containerName := containerPrefix + gameName

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("container %s not found", containerName)
	}

	c := containers[0]

	// Stop if running
	if c.State == "running" {
		if err := cli.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Remove container
	return cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{
		Force: true,
	})
}

// RemoveImage removes the Docker image for a game
func RemoveImage(imageName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	_, err = cli.ImageRemove(ctx, imageName, image.RemoveOptions{
		Force: true,
	})
	return err
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
