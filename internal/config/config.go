package config

import (
	"os"
	"path/filepath"
)

const (
	appName     = "hostathome"
	cacheSubdir = "cache/registry"
)

// GetCacheDir returns the cache directory for registry files
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, "."+appName, cacheSubdir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// GetConfigDir returns the main config directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, "."+appName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}
