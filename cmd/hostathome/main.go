package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hostathome/cli/internal/docker"
	"github.com/hostathome/cli/internal/registry"
	"github.com/hostathome/cli/internal/ui"
	"github.com/hostathome/cli/internal/version"
	"github.com/spf13/cobra"
)

var cliVersion = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "hostathome",
	Short: "Manage game servers with ease",
	Long: `HostAtHome CLI - Install, run, and manage game servers using Docker.

Run 'hostathome doctor' to check if your system is ready.
Run 'hostathome list' to see available games.`,
	Version: cliVersion,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip update check for update command itself
		if cmd.Name() == "update" || cmd.Name() == "version" {
			return
		}

		// Check for updates (once per day max)
		if hasUpdate, latest := version.CheckForUpdate(cliVersion); hasUpdate {
			fmt.Println()
			ui.Info("A new version of HostAtHome CLI is available: v%s (current: v%s)", latest, cliVersion)
			ui.Detail("Update", "Run 'hostathome update' to upgrade")
			fmt.Println()
		}
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system requirements",
	Long:  "Verify that Docker is installed and running, and check system readiness.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Title("HostAtHome Doctor")
		fmt.Println()

		allGood := true

		// Check Docker installed
		ui.Step("Checking Docker installation...")
		_, err := exec.LookPath("docker")
		if err != nil {
			ui.Error("Docker not found in PATH")
			ui.Detail("Fix", "Install Docker: https://docs.docker.com/get-docker/")
			allGood = false
		} else {
			ui.Success("Docker is installed")
		}

		// Check Docker daemon running
		ui.Step("Checking Docker daemon...")
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			ui.Error("Cannot connect to Docker: %v", err)
			allGood = false
		} else {
			defer cli.Close()
			_, err = cli.Ping(context.Background())
			if err != nil {
				ui.Error("Docker daemon not running")
				ui.Detail("Fix", "Start Docker: sudo systemctl start docker")
				allGood = false
			} else {
				ui.Success("Docker daemon is running")
			}
		}

		// Check Docker permissions
		ui.Step("Checking Docker permissions...")
		ctx := context.Background()
		if cli != nil {
			_, err = cli.ImageList(ctx, image.ListOptions{})
			if err != nil {
				ui.Error("Cannot access Docker (permission denied?)")
				ui.Detail("Fix", "Add user to docker group: sudo usermod -aG docker $USER")
				ui.Detail("Note", "Log out and back in after adding to group")
				allGood = false
			} else {
				ui.Success("Docker permissions OK")
			}
		}

		// Check registry access
		ui.Step("Checking registry access...")
		_, err = registry.ListGames()
		if err != nil {
			ui.Warning("Cannot fetch game registry (offline?)")
			ui.Detail("Note", "CLI will use cached data if available")
		} else {
			ui.Success("Registry accessible")
		}

		fmt.Println()
		if allGood {
			ui.Success("All checks passed! You're ready to go.")
			fmt.Println()
			ui.Info("Try: hostathome list")
		} else {
			ui.Error("Some checks failed. Please fix the issues above.")
			return fmt.Errorf("system not ready")
		}
		return nil
	},
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
			ui.Error("Game '%s' not found", gameName)
			ui.Info("Run 'hostathome list' to see available games")
			return err
		}

		ui.Title("Installing %s", game.DisplayName)
		fmt.Println()

		// Pull Docker image
		spinner := ui.NewSpinner(fmt.Sprintf("Pulling %s", game.Image))
		spinner.Start()
		if err := docker.PullImage(game.Image); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to pull image: %w", err)
		}
		spinner.Stop(true)

		// Create directory structure
		spinner = ui.NewSpinner("Creating directory structure")
		spinner.Start()
		if err := docker.CreateServerDirs(gameName); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to create directories: %w", err)
		}
		spinner.Stop(true)

		// Copy default config
		spinner = ui.NewSpinner("Writing default configuration")
		spinner.Start()
		if err := registry.CopyDefaultConfig(gameName, game); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to copy default config: %w", err)
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Success("%s installed successfully!", game.DisplayName)
		fmt.Println()
		ui.Detail("Directory", fmt.Sprintf("./%s-server/", gameName))
		ui.Detail("Config", fmt.Sprintf("./%s-server/configs/config.yaml", gameName))
		fmt.Println()
		ui.Info("Start with: hostathome run %s", gameName)

		return nil
	},
}

var devMode bool

var runCmd = &cobra.Command{
	Use:   "run <game>",
	Short: "Start a game server",
	Long:  "Start the game server container.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		var game *registry.Game
		var err error

		if devMode {
			// Dev mode: use local :dev image
			game = &registry.Game{
				Name:        gameName,
				DisplayName: gameName + " (dev)",
				Image:       gameName + "-server:dev",
				Ports: registry.Ports{
					Player: 1024,
					RCON:   1025,
				},
				InternalPorts: registry.Ports{
					Player: 25565,
					RCON:   25575,
				},
				Protocols: registry.Protocols{
					Player: "tcp",
					RCON:   "tcp",
				},
			}
			fmt.Println("🔧 Development mode: using local image", game.Image)
		} else {
			// Normal mode: fetch from registry
			game, err = registry.GetGame(gameName)
			if err != nil {
				ui.Error("Game '%s' not found", gameName)
				return err
			}
		}

		spinner := ui.NewSpinner(fmt.Sprintf("Starting %s", game.DisplayName))
		spinner.Start()

		if err := docker.RunContainer(gameName, game, devMode); err != nil {
			spinner.Stop(false)
			ui.Error("Failed to start container: %v", err)
			return fmt.Errorf("failed to start container: %w", err)
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Success("%s is running!", game.DisplayName)
		fmt.Println()
		ui.Detail("Player port", fmt.Sprintf("%d", game.Ports.Player))
		if game.Ports.RCON > 0 {
			ui.Detail("RCON port", fmt.Sprintf("%d", game.Ports.RCON))
		}
		fmt.Println()
		ui.Info("View logs: hostathome logs %s", gameName)
		ui.Info("Stop: hostathome stop %s", gameName)

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
			ui.Error("Game '%s' not found", gameName)
			return err
		}

		spinner := ui.NewSpinner(fmt.Sprintf("Stopping %s", game.DisplayName))
		spinner.Start()

		if err := docker.StopContainer(gameName); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to stop container: %w", err)
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Success("%s stopped.", game.DisplayName)

		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart <game>",
	Short: "Restart a game server",
	Long:  "Restart the game server container to apply configuration changes.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		game, err := registry.GetGame(gameName)
		if err != nil {
			ui.Error("Game '%s' not found", gameName)
			return err
		}

		spinner := ui.NewSpinner(fmt.Sprintf("Restarting %s", game.DisplayName))
		spinner.Start()

		if err := docker.RestartContainer(gameName); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to restart container: %w", err)
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Success("%s restarted.", game.DisplayName)
		ui.Info("Configuration changes have been applied")

		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <game>",
	Short: "Remove a game server container",
	Long:  "Remove the game server container but keep the data directory.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		game, err := registry.GetGame(gameName)
		if err != nil {
			ui.Error("Game '%s' not found", gameName)
			return err
		}

		spinner := ui.NewSpinner(fmt.Sprintf("Removing %s container", game.DisplayName))
		spinner.Start()

		if err := docker.RemoveContainer(gameName); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to remove container: %w", err)
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Success("%s container removed.", game.DisplayName)
		fmt.Println()
		ui.Detail("Data preserved", fmt.Sprintf("./%s-server/", gameName))
		ui.Info("Run 'hostathome run %s' to recreate the container", gameName)

		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <game>",
	Short: "Uninstall a game server completely",
	Long:  "Remove the container, Docker image, and data directory.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		game, err := registry.GetGame(gameName)
		if err != nil {
			ui.Error("Game '%s' not found", gameName)
			return err
		}

		// Confirm before deleting data
		fmt.Println()
		ui.Warning("This will permanently delete all data for %s", game.DisplayName)
		ui.Detail("Directory", fmt.Sprintf("./%s-server/", gameName))
		fmt.Println()
		fmt.Print("Are you sure? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			ui.Info("Cancelled.")
			return nil
		}

		// Remove container
		spinner := ui.NewSpinner(fmt.Sprintf("Removing %s container", game.DisplayName))
		spinner.Start()
		if err := docker.RemoveContainer(gameName); err != nil {
			// Container might not exist, that's ok
			spinner.StopWithMessage(true, fmt.Sprintf("No container found for %s", game.DisplayName))
		} else {
			spinner.Stop(true)
		}

		// Remove image
		spinner = ui.NewSpinner(fmt.Sprintf("Removing %s image", game.Image))
		spinner.Start()
		if err := docker.RemoveImage(game.Image); err != nil {
			spinner.StopWithMessage(true, "Image not found (may be in use by other containers)")
		} else {
			spinner.Stop(true)
		}

		// Remove data directory
		spinner = ui.NewSpinner("Removing data directory")
		spinner.Start()
		dataDir := fmt.Sprintf("./%s-server", gameName)
		if err := os.RemoveAll(dataDir); err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to remove data directory: %w", err)
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Success("%s uninstalled completely.", game.DisplayName)

		return nil
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs <game>",
	Short: "View server logs",
	Long:  "Stream logs from the game server container.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]
		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetString("tail")

		containerName := "hostathome-" + gameName

		cmdArgs := []string{"logs"}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}
		if tail != "" {
			cmdArgs = append(cmdArgs, "--tail", tail)
		}
		cmdArgs = append(cmdArgs, containerName)

		dockerCmd := exec.Command("docker", cmdArgs...)
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr

		return dockerCmd.Run()
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update HostAtHome CLI to the latest version",
	Long:  "Download and install the latest version of HostAtHome CLI from GitHub releases.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Title("HostAtHome CLI Update")
		fmt.Println()

		// Check for updates
		spinner := ui.NewSpinner("Checking for updates")
		spinner.Start()

		latest, err := version.GetLatestVersion()
		if err != nil {
			spinner.Stop(false)
			return fmt.Errorf("failed to check for updates: %w", err)
		}
		spinner.Stop(true)

		// Compare versions
		if !version.CompareVersions(cliVersion, latest) {
			ui.Success("You're already running the latest version (v%s)", cliVersion)
			return nil
		}

		fmt.Println()
		ui.Info("New version available: v%s (current: v%s)", latest, cliVersion)
		fmt.Println()

		// Confirm update
		fmt.Print("Do you want to update? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" && response != "y" {
			ui.Info("Update cancelled.")
			return nil
		}

		fmt.Println()

		// Perform update
		if err := version.Update(); err != nil {
			ui.Error("Update failed: %v", err)
			return err
		}

		fmt.Println()
		ui.Success("HostAtHome CLI updated to v%s!", latest)
		ui.Info("Run 'hostathome --version' to verify")

		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config <game>",
	Short: "Edit server configuration",
	Long:  "Open the server configuration file in your default editor.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		configPath := filepath.Join(fmt.Sprintf("./%s-server", gameName), "configs", "config.yaml")

		// Check if config exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			ui.Error("Config not found: %s", configPath)
			ui.Info("Run 'hostathome install %s' first", gameName)
			return err
		}

		// Get editor from env
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			// Try common editors
			for _, e := range []string{"nano", "vim", "vi"} {
				if _, err := exec.LookPath(e); err == nil {
					editor = e
					break
				}
			}
		}
		if editor == "" {
			ui.Error("No editor found")
			ui.Detail("Fix", "Set EDITOR environment variable")
			return fmt.Errorf("no editor found")
		}

		ui.Info("Opening %s with %s...", configPath, editor)

		editorCmd := exec.Command(editor, configPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return err
		}

		// Prompt to restart
		fmt.Println()
		fmt.Print("Restart server to apply changes? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response == "yes" || response == "y" {
			game, err := registry.GetGame(gameName)
			if err != nil {
				return err
			}

			spinner := ui.NewSpinner(fmt.Sprintf("Restarting %s", game.DisplayName))
			spinner.Start()
			if err := docker.RestartContainer(gameName); err != nil {
				spinner.Stop(false)
				ui.Warning("Failed to restart. Run 'hostathome restart %s' manually", gameName)
			} else {
				spinner.Stop(true)
				ui.Success("Configuration applied!")
			}
		}

		return nil
	},
}

var modsCmd = &cobra.Command{
	Use:   "mods <game>",
	Short: "Edit mods configuration",
	Long:  "Open the mods configuration file in your default editor.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gameName := args[0]

		modsPath := filepath.Join(fmt.Sprintf("./%s-server", gameName), "mods", "mods.yaml")

		// Check if mods config exists
		if _, err := os.Stat(modsPath); os.IsNotExist(err) {
			ui.Error("Mods config not found: %s", modsPath)
			ui.Info("Run 'hostathome install %s' first", gameName)
			return err
		}

		// Get editor from env
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			// Try common editors
			for _, e := range []string{"nano", "vim", "vi"} {
				if _, err := exec.LookPath(e); err == nil {
					editor = e
					break
				}
			}
		}
		if editor == "" {
			ui.Error("No editor found")
			ui.Detail("Fix", "Set EDITOR environment variable")
			return fmt.Errorf("no editor found")
		}

		ui.Info("Opening %s with %s...", modsPath, editor)

		editorCmd := exec.Command(editor, modsPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return err
		}

		// Prompt to restart
		fmt.Println()
		fmt.Print("Restart server to load mods? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response == "yes" || response == "y" {
			game, err := registry.GetGame(gameName)
			if err != nil {
				return err
			}

			spinner := ui.NewSpinner(fmt.Sprintf("Restarting %s", game.DisplayName))
			spinner.Start()
			if err := docker.RestartContainer(gameName); err != nil {
				spinner.Stop(false)
				ui.Warning("Failed to restart. Run 'hostathome restart %s' manually", gameName)
			} else {
				spinner.Stop(true)
				ui.Success("Mods configuration applied!")
			}
		}

		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [game]",
	Short: "Show server status",
	Long:  "Show the status of running game servers.",
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
				ui.Info("No container found for %s", gameName)
			} else {
				ui.Info("No HostAtHome containers running")
			}
			fmt.Println()
			ui.Info("Install a game: hostathome install <game>")
			ui.Info("List games: hostathome list")
			return nil
		}

		ui.Title("Server Status")
		fmt.Println()

		headers := []string{"GAME", "STATUS", "PORTS", "CONTAINER"}
		var rows [][]string
		for _, s := range statuses {
			status := s.Status
			if s.Status == "running" {
				status = ui.SymbolCheck + " running"
			} else if s.Status == "exited" {
				status = ui.SymbolCross + " stopped"
			}
			rows = append(rows, []string{s.Game, status, s.Ports, s.ContainerID[:12]})
		}
		ui.Table(headers, rows)

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available games",
	Long:  "Show all games available in the registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner := ui.NewSpinner("Fetching game list")
		spinner.Start()

		games, err := registry.ListGames()
		if err != nil {
			spinner.Stop(false)
			return err
		}
		spinner.Stop(true)

		fmt.Println()
		ui.Title("Available Games")
		fmt.Println()

		headers := []string{"GAME", "DESCRIPTION"}
		var rows [][]string
		for _, g := range games {
			rows = append(rows, []string{g.Name, g.Description})
		}
		ui.Table(headers, rows)

		fmt.Println()
		ui.Info("Install: hostathome install <game>")

		return nil
	},
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().StringP("tail", "n", "100", "Number of lines to show")

	runCmd.Flags().BoolVarP(&devMode, "dev", "d", false, "Use local dev image instead of registry")

	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(modsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(updateCmd)
}
