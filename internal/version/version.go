package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	githubAPIURL = "https://api.github.com/repos/hostathome/cli/releases/latest"
	checkInterval = 24 * time.Hour
	cacheFile = ".hostathome/cache/version_check"
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// getCacheFile returns the full path to the version check cache file
func getCacheFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/" + cacheFile
}

// shouldCheckForUpdate checks if we should query GitHub (rate limiting)
func shouldCheckForUpdate() bool {
	cacheFile := getCacheFile()
	if cacheFile == "" {
		return false
	}

	info, err := os.Stat(cacheFile)
	if err != nil {
		return true // No cache file, should check
	}

	return time.Since(info.ModTime()) > checkInterval
}

// markChecked creates/updates the cache file timestamp
func markChecked() {
	cacheFile := getCacheFile()
	if cacheFile == "" {
		return
	}

	// Ensure cache directory exists
	os.MkdirAll(strings.TrimSuffix(cacheFile, "/version_check"), 0755)
	os.WriteFile(cacheFile, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// GetLatestVersion fetches the latest version from GitHub
func GetLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(githubAPIURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch latest version: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// CompareVersions returns true if latest > current
func CompareVersions(current, latest string) bool {
	// Remove 'v' prefix if present
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// Handle dev version
	if current == "dev" {
		return false
	}

	// Simple semantic version comparison (major.minor.patch)
	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	for i := 0; i < 3; i++ {
		var cv, lv int
		if i < len(currentParts) {
			fmt.Sscanf(currentParts[i], "%d", &cv)
		}
		if i < len(latestParts) {
			fmt.Sscanf(latestParts[i], "%d", &lv)
		}

		if lv > cv {
			return true
		} else if lv < cv {
			return false
		}
	}

	return false
}

// CheckForUpdate checks if a new version is available (with rate limiting)
func CheckForUpdate(currentVersion string) (hasUpdate bool, latestVersion string) {
	// Skip if we checked recently
	if !shouldCheckForUpdate() {
		return false, ""
	}

	latest, err := GetLatestVersion()
	if err != nil {
		// Silently fail - don't bother users with network errors
		return false, ""
	}

	markChecked()

	if CompareVersions(currentVersion, latest) {
		return true, latest
	}

	return false, ""
}

// Update performs a self-update by downloading and installing the new .deb
func Update() error {
	// Determine architecture
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	} else {
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Use /latest/ endpoint with version-less filename
	debFile := fmt.Sprintf("hostathome_%s.deb", arch)
	downloadURL := fmt.Sprintf("https://github.com/hostathome/cli/releases/latest/download/%s", debFile)

	// Download to /tmp
	tmpFile := "/tmp/" + debFile
	fmt.Printf("Downloading %s...\n", downloadURL)

	cmd := exec.Command("wget", "-O", tmpFile, downloadURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Install with dpkg
	fmt.Println("Installing update...")
	installCmd := exec.Command("sudo", "dpkg", "-i", tmpFile)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Clean up
	os.Remove(tmpFile)

	return nil
}
