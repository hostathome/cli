package registry

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/hostathome/cli/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	registryBaseURL   = "https://raw.githubusercontent.com/hostathome/registry/main"
	registryGamesURL  = registryBaseURL + "/games"
	cacheTTL          = 1 * time.Hour
	httpTimeout       = 30  // seconds
	dockerOpTimeout   = 30  // seconds
)

var gameCache = make(map[string]*Game)

// validateGameName checks if gameName is valid for use in paths
func validateGameName(gameName string) error {
	if gameName == "" {
		return fmt.Errorf("game name cannot be empty")
	}
	if len(gameName) > 63 {
		return fmt.Errorf("game name too long (max 63 chars)")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(gameName) {
		return fmt.Errorf("game name contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}
	if strings.Contains(gameName, "..") || strings.HasPrefix(gameName, "/") || strings.HasPrefix(gameName, ".") {
		return fmt.Errorf("game name cannot contain path traversal sequences")
	}
	return nil
}

// getHTTPClient returns an HTTP client with timeout
func getHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Duration(httpTimeout) * time.Second,
	}
}

// getCacheDir returns the full path to the cache directory
func getCacheDir() string {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return ""
	}
	return cacheDir
}

// GetGame returns a game definition by name
func GetGame(name string) (*Game, error) {
	if game, ok := gameCache[name]; ok {
		return game, nil
	}

	data, err := fetchWithCache(name)
	if err != nil {
		return nil, fmt.Errorf("game '%s' not found in registry: %w", name, err)
	}

	var game Game
	if err := yaml.Unmarshal(data, &game); err != nil {
		return nil, fmt.Errorf("failed to parse game definition: %w", err)
	}

	gameCache[name] = &game
	return &game, nil
}

// fetchWithCache fetches a game definition from GitHub or cache
func fetchWithCache(name string) ([]byte, error) {
	cacheFile := filepath.Join(getCacheDir(), name+".yaml")

	// Check if cache exists and is fresh
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < cacheTTL {
			return os.ReadFile(cacheFile)
		}
	}

	// Fetch from GitHub
	url := fmt.Sprintf("%s/%s.yaml", registryGamesURL, name)
	resp, err := getHTTPClient().Get(url)
	if err != nil {
		// Fall back to stale cache if available
		if data, cacheErr := os.ReadFile(cacheFile); cacheErr == nil {
			return data, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("game not found")
	}
	if resp.StatusCode != 200 {
		// Fall back to stale cache
		if data, cacheErr := os.ReadFile(cacheFile); cacheErr == nil {
			return data, nil
		}
		return nil, fmt.Errorf("failed to fetch: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Save to cache (non-critical, don't fail if it fails)
	if dir := getCacheDir(); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Log but don't fail - cache is optional
			_ = err
		} else if err := os.WriteFile(cacheFile, data, 0644); err != nil {
			// Log but don't fail - cache is optional
			_ = err
		}
	}

	return data, nil
}

// ListGames returns all available games
func ListGames() ([]Game, error) {
	index, err := fetchGameIndex()
	if err != nil {
		return nil, err
	}

	var games []Game
	for _, name := range index {
		game, err := GetGame(name)
		if err != nil {
			continue
		}
		games = append(games, *game)
	}
	return games, nil
}

// fetchGameIndex fetches the list of available games
func fetchGameIndex() ([]string, error) {
	cacheFile := filepath.Join(getCacheDir(), "index.json")

	// Check cache
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < cacheTTL {
			if data, err := os.ReadFile(cacheFile); err == nil {
				var index []string
				if err := json.Unmarshal(data, &index); err == nil && len(index) > 0 {
					return index, nil
				}
			}
		}
	}

	// Fetch index.yaml from GitHub
	url := registryBaseURL + "/index.yaml"
	resp, err := getHTTPClient().Get(url)
	if err != nil {
		// Fall back to cache if available
		if data, cacheErr := os.ReadFile(cacheFile); cacheErr == nil && len(data) > 0 {
			var index []string
			if cacheParseErr := json.Unmarshal(data, &index); cacheParseErr == nil && len(index) > 0 {
				return index, nil
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Fall back to cache if available
		if data, cacheErr := os.ReadFile(cacheFile); cacheErr == nil && len(data) > 0 {
			var index []string
			if cacheParseErr := json.Unmarshal(data, &index); cacheParseErr == nil && len(index) > 0 {
				return index, nil
			}
		}
		return nil, fmt.Errorf("failed to fetch game index: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var indexFile struct {
		Games []string `yaml:"games"`
	}
	if err := yaml.Unmarshal(data, &indexFile); err != nil {
		return nil, err
	}

	// Cache as JSON (non-critical, don't fail if it fails)
	if dir := getCacheDir(); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			_ = err // Log but don't fail - cache is optional
		} else if jsonData, err := json.Marshal(indexFile.Games); err == nil {
			_ = os.WriteFile(cacheFile, jsonData, 0644) // Log but don't fail
		}
	}

	return indexFile.Games, nil
}

// CopyDefaultConfig extracts default configs from the Docker image
func CopyDefaultConfig(gameName string, game *Game) error {
	if err := validateGameName(gameName); err != nil {
		return fmt.Errorf("invalid game name: %w", err)
	}

	serverDir := fmt.Sprintf("./%s-server", gameName)
	configDir := filepath.Join(serverDir, "configs")

	// Create configs directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Extract configs from the Docker image
	ctx, cancel := context.WithTimeout(context.Background(), dockerOpTimeout*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer cli.Close()

	// Copy config.yaml if it doesn't exist
	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		if err := extractFileFromImage(ctx, cli, game.Image, "/defaults/config.yaml", configPath); err != nil {
			return fmt.Errorf("failed to extract config: %w", err)
		}
	}

	// Copy mods.yaml if it doesn't exist
	modsPath := filepath.Join(configDir, "mods.yaml")
	if _, err := os.Stat(modsPath); err != nil {
		if err := extractFileFromImage(ctx, cli, game.Image, "/defaults/mods.yaml", modsPath); err != nil {
			// mods.yaml is optional, don't fail if extraction fails
		}
	}

	return nil
}

// extractFileFromImage uses a temporary container to extract files from a Docker image
func extractFileFromImage(ctx context.Context, cli *client.Client, imageRef, containerPath, destPath string) error {
	// Create a temporary container
	resp, err := cli.ContainerCreate(ctx, &container.Config{Image: imageRef}, nil, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create temporary container: %w", err)
	}
	defer func() {
		// Clean up temporary container (use docker CLI to avoid depending on a specific SDK type)
		_ = exec.Command("docker", "rm", "-f", resp.ID).Run()
	}()

	// Copy file from container
	readCloser, _, err := cli.CopyFromContainer(ctx, resp.ID, containerPath)
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	defer readCloser.Close()

	// Extract from tar archive
	// When copying /path/to/file, the tar archive contains just the filename
	expectedName := filepath.Base(containerPath)
	tr := tar.NewReader(readCloser)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("file %s not found in image", containerPath)
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Match the filename in the tar archive
		if header.Name == expectedName {
			// Write the file
			file, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(file, tr); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			// Set proper permissions (user read/write, group/other readable)
			if err := os.Chmod(destPath, 0644); err != nil {
				return fmt.Errorf("failed to set file permissions: %w", err)
			}

			return nil
		}
	}
}

