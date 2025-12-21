package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	registryBaseURL = "https://raw.githubusercontent.com/hostathome/registry/main/games"
	cacheDir        = ".hostathome/cache/registry"
	cacheTTL        = 1 * time.Hour
)

var gameCache = make(map[string]*Game)

// getCacheDir returns the full path to the cache directory
func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, cacheDir)
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
	url := fmt.Sprintf("%s/%s.yaml", registryBaseURL, name)
	resp, err := http.Get(url)
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

	// Save to cache
	if dir := getCacheDir(); dir != "" {
		os.MkdirAll(dir, 0755)
		os.WriteFile(cacheFile, data, 0644)
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
			data, _ := os.ReadFile(cacheFile)
			var index []string
			if json.Unmarshal(data, &index) == nil {
				return index, nil
			}
		}
	}

	// Fetch index.yaml from GitHub
	url := fmt.Sprintf("%s/../index.yaml", registryBaseURL)
	resp, err := http.Get(url)
	if err != nil {
		// Fall back to cache
		if data, _ := os.ReadFile(cacheFile); len(data) > 0 {
			var index []string
			json.Unmarshal(data, &index)
			return index, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Fall back to cache
		if data, _ := os.ReadFile(cacheFile); len(data) > 0 {
			var index []string
			json.Unmarshal(data, &index)
			return index, nil
		}
		return nil, fmt.Errorf("failed to fetch game index: %s", resp.Status)
	}

	data, _ := io.ReadAll(resp.Body)

	var indexFile struct {
		Games []string `yaml:"games"`
	}
	if err := yaml.Unmarshal(data, &indexFile); err != nil {
		return nil, err
	}

	// Cache as JSON
	if dir := getCacheDir(); dir != "" {
		os.MkdirAll(dir, 0755)
		jsonData, _ := json.Marshal(indexFile.Games)
		os.WriteFile(cacheFile, jsonData, 0644)
	}

	return indexFile.Games, nil
}

// CopyDefaultConfig copies the default config.yaml if it doesn't exist
func CopyDefaultConfig(gameName string, game *Game) error {
	configDir := fmt.Sprintf("./%s-server/configs", gameName)
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Already exists
	}

	// Generate default config from schema
	defaultConfig := generateDefaultConfig(game.ConfigSchema)

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}

	// Add header comment
	header := fmt.Sprintf("# %s Server Configuration\n\n", game.DisplayName)
	return os.WriteFile(configPath, append([]byte(header), data...), 0644)
}

// generateDefaultConfig creates a config map from the schema with defaults
func generateDefaultConfig(schema map[string]any) map[string]any {
	config := make(map[string]any)

	for section, fields := range schema {
		sectionMap := make(map[string]any)
		if fieldMap, ok := fields.(map[string]any); ok {
			for field, spec := range fieldMap {
				if specMap, ok := spec.(map[string]any); ok {
					if def, exists := specMap["default"]; exists {
						sectionMap[field] = def
					}
				}
			}
		}
		if len(sectionMap) > 0 {
			config[section] = sectionMap
		}
	}

	return config
}
