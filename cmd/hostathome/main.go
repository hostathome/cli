package main

import (
	"fmt"
	"os"

	"github.com/hostathome/cli/internal/docker"
	"github.com/hostathome/cli/internal/registry"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "hostathome",
	Short:   "Manage game servers with ease",
	Long:    "HostAtHome CLI - Install, run, and manage game servers using Docker.",
	Version: version,
}

var installCmd = &cobra.Command{
	Use:   "install <game>",
	Short: "Install a game server",
	Long:  "Pull the Docker image and create the server directory structure.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		game, err := registry.GetGame(gameName)
		if err != nil {
			return err
		}

		fmt.Printf("Installing %s...\n", game.DisplayName)

		// Pull Docker image
		if err := docker.PullImage(game.Image); err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}

		// Create directory structure
		if err := docker.CreateServerDirs(gameName); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		// Copy default config if it doesn't exist
		if err := registry.CopyDefaultConfig(gameName, game); err != nil {
			return fmt.Errorf("failed to copy default config: %w", err)
		}

		fmt.Printf("%s installed successfully!\n", game.DisplayName)
		fmt.Printf("  Directory: ./%s-server/\n", gameName)
		fmt.Printf("  Config:    ./%s-server/configs/config.yaml\n", gameName)
		fmt.Printf("\nRun with: hostathome run %s\n", gameName)
		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run <game>",
	Short: "Start a game server",
	Long:  "Start the game server container (or restart if already running).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		game, err := registry.GetGame(gameName)
		if err != nil {
			return err
		}

		fmt.Printf("Starting %s...\n", game.DisplayName)

		if err := docker.RunContainer(gameName, game); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}

		fmt.Printf("%s is running!\n", game.DisplayName)
		fmt.Printf("  Player port: %d\n", game.Ports.Player)
		if game.Ports.RCON > 0 {
			fmt.Printf("  RCON port:   %d\n", game.Ports.RCON)
		}
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <game>",
	Short: "Stop a game server",
	Long:  "Stop the running game server container.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		game, err := registry.GetGame(gameName)
		if err != nil {
			return err
		}

		fmt.Printf("Stopping %s...\n", game.DisplayName)

		if err := docker.StopContainer(gameName); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		fmt.Printf("%s stopped.\n", game.DisplayName)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [game]",
	Short: "Show server status",
	Long:  "Show the status of running game servers. Optionally specify a game.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var gameName string
		if len(args) > 0 {
			gameName = args[0]
		}

		statuses, err := docker.GetStatus(gameName)
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		if len(statuses) == 0 {
			if gameName != "" {
				fmt.Printf("No container found for %s\n", gameName)
			} else {
				fmt.Println("No HostAtHome containers running")
			}
			return nil
		}

		fmt.Printf("%-15s %-10s %-20s %s\n", "GAME", "STATUS", "PORTS", "CONTAINER")
		fmt.Printf("%-15s %-10s %-20s %s\n", "----", "------", "-----", "---------")
		for _, s := range statuses {
			fmt.Printf("%-15s %-10s %-20s %s\n", s.Game, s.Status, s.Ports, s.ContainerID[:12])
		}
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available games",
	Long:  "Show all games available in the registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		games, err := registry.ListGames()
		if err != nil {
			return err
		}

		fmt.Printf("%-15s %s\n", "GAME", "DESCRIPTION")
		fmt.Printf("%-15s %s\n", "----", "-----------")
		for _, g := range games {
			fmt.Printf("%-15s %s\n", g.Name, g.Description)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
}
