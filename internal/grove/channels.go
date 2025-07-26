package grove

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// NixOSRelease represents a NixOS release from the GitHub API.
type NixOSRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Draft      bool   `json:"draft"`
	PreRelease bool   `json:"prerelease"`
}

// ChannelInfo represents nixpkgs channel information.
type ChannelInfo struct {
	Name        string
	URL         string
	Description string
}

// ChannelConfig represents a channel configuration from devenv.yaml
type ChannelConfig struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

// DevenvYamlInputs represents the inputs section of devenv.yaml
type DevenvYamlInputs struct {
	URL string `yaml:"url"`
}

// DevenvYaml represents the structure of devenv.yaml
type DevenvYaml struct {
	Inputs      map[string]DevenvYamlInputs `yaml:"inputs"`
	AllowUnfree bool                        `yaml:"allowUnfree,omitempty"`
	Backend     string                      `yaml:"backend,omitempty"`
}

const (
	// Fallback stable channel if API fails.
	FallbackStableChannel = "nixos-24.05"
	// NixOS channels API endpoint (more reliable for stable releases).
	NixOSChannelsAPI = "https://channels.nixos.org/"
	// GitHub API endpoint for NixOS releases (backup).
	NixOSReleasesAPI = "https://api.github.com/repos/NixOS/nixpkgs/releases"
	// HTTP timeout for API calls.
	APITimeout = 10 * time.Second
)

// GetLatestStableChannel fetches the latest stable NixOS channel programmatically.
func GetLatestStableChannel() string {
	// Try multiple methods to get the latest stable channel

	// Method 1: Try to parse from known stable channels
	latest, err := fetchLatestStableFromKnownPattern()
	if err == nil {
		return latest
	}

	// Method 2: Try GitHub releases API (backup)
	latest, err = fetchLatestStableRelease()
	if err == nil {
		return latest
	}

	// Fallback: Use known good version
	return FallbackStableChannel
}

// fetchLatestStableFromKnownPattern determines the latest stable channel using date-based logic.
func fetchLatestStableFromKnownPattern() (string, error) {
	// NixOS releases stable versions twice a year: .05 (May) and .11 (November)
	// We can determine the latest stable based on current date

	now := time.Now()
	currentYear := now.Year() % 100 // Get last 2 digits of year
	currentMonth := int(now.Month())

	var latestYear, latestMonth int

	if currentMonth >= 11 {
		// After November, the latest stable is YY.11
		latestYear = currentYear
		latestMonth = 11
	} else if currentMonth >= 5 {
		// Between May and October, the latest stable is YY.05
		latestYear = currentYear
		latestMonth = 5
	} else {
		// Before May, the latest stable is from previous year's November
		latestYear = currentYear - 1
		latestMonth = 11

		// Handle year rollover (e.g., 2024 -> 2023 becomes 24 -> 23)
		if latestYear < 0 {
			latestYear = 99
		}
	}

	// Format as nixos-YY.MM
	channelName := fmt.Sprintf("nixos-%02d.%02d", latestYear, latestMonth)

	// Verify this channel exists by making a quick HTTP request
	if verifyChannelExists(channelName) {
		return channelName, nil
	}

	return "", fmt.Errorf("calculated channel %s does not exist", channelName)
}

// verifyChannelExists checks if a NixOS channel exists.
func verifyChannelExists(channelName string) bool {
	// Try to access the channel URL to verify it exists
	channelURL := fmt.Sprintf("https://channels.nixos.org/%s", channelName)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Head(channelURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Channel exists if we get a 200 or 302 response
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound
}

// fetchLatestStableRelease queries GitHub API for the latest stable NixOS release.
func fetchLatestStableRelease() (string, error) {
	client := &http.Client{
		Timeout: APITimeout,
	}

	resp, err := client.Get(NixOSReleasesAPI)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var releases []NixOSRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to decode releases: %w", err)
	}

	// Filter for stable releases and find the latest
	stableReleases := filterStableReleases(releases)
	if len(stableReleases) == 0 {
		return "", fmt.Errorf("no stable releases found")
	}

	// Sort by version number (descending) and return the latest
	sort.Slice(stableReleases, func(i, j int) bool {
		return compareVersions(stableReleases[i].TagName, stableReleases[j].TagName) > 0
	})

	latestTag := stableReleases[0].TagName

	// Convert tag to channel format (e.g., "23.11" -> "nixos-23.11")
	channelName := tagToChannelName(latestTag)

	return channelName, nil
}

// filterStableReleases filters releases to only include stable NixOS releases.
func filterStableReleases(releases []NixOSRelease) []NixOSRelease {
	var stable []NixOSRelease

	for _, release := range releases {
		// Skip drafts and pre-releases
		if release.Draft || release.PreRelease {
			continue
		}

		// Only include releases that match NixOS stable pattern (e.g., "23.11", "24.05")
		if isStableReleaseTag(release.TagName) {
			stable = append(stable, release)
		}
	}

	return stable
}

// isStableReleaseTag checks if a tag represents a stable NixOS release.
func isStableReleaseTag(tag string) bool {
	// NixOS stable releases follow the pattern: YY.MM (e.g., "23.11", "24.05")
	parts := strings.Split(tag, ".")
	if len(parts) != 2 {
		return false
	}

	// Check if both parts are numeric
	year, err1 := strconv.Atoi(parts[0])
	month, err2 := strconv.Atoi(parts[1])

	if err1 != nil || err2 != nil {
		return false
	}

	// Validate reasonable ranges
	if year < 20 || year > 99 || month < 1 || month > 12 {
		return false
	}

	return true
}

// tagToChannelName converts a release tag to nixpkgs channel name.
func tagToChannelName(tag string) string {
	// Convert "23.11" to "nixos-23.11"
	return "nixos-" + tag
}

// compareVersions compares two version strings (e.g., "23.11" vs "24.05")
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal.
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	if len(parts1) != 2 || len(parts2) != 2 {
		return strings.Compare(v1, v2)
	}

	year1, _ := strconv.Atoi(parts1[0])
	month1, _ := strconv.Atoi(parts1[1])
	year2, _ := strconv.Atoi(parts2[0])
	month2, _ := strconv.Atoi(parts2[1])

	if year1 != year2 {
		if year1 > year2 {
			return 1
		}
		return -1
	}

	if month1 > month2 {
		return 1
	} else if month1 < month2 {
		return -1
	}

	return 0
}

// GetDefaultChannels returns the default channel configuration.
func GetDefaultChannels() map[string]ChannelInfo {
	latestStable := GetLatestStableChannel()

	return map[string]ChannelInfo{
		"nixpkgs": {
			Name:        "nixpkgs",
			URL:         "github:NixOS/nixpkgs/nixpkgs-unstable",
			Description: "Latest unstable packages",
		},
		"nixpkgs-stable": {
			Name:        "nixpkgs-stable",
			URL:         fmt.Sprintf("github:NixOS/nixpkgs/%s", latestStable),
			Description: fmt.Sprintf("Latest stable packages (%s)", latestStable),
		},
	}
}

// ListChannels returns all channels configured in devenv.yaml
func ListChannels() ([]ChannelConfig, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")
	
	// Check if devenv.yaml exists
	if _, err := os.Stat(devenvYamlPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("devenv.yaml not found")
	}

	// Read and parse devenv.yaml
	content, err := os.ReadFile(devenvYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read devenv.yaml: %w", err)
	}

	var devenvConfig DevenvYaml
	if err := yaml.Unmarshal(content, &devenvConfig); err != nil {
		return nil, fmt.Errorf("failed to parse devenv.yaml: %w", err)
	}

	// Convert inputs to ChannelConfig slice
	var channels []ChannelConfig
	for name, input := range devenvConfig.Inputs {
		description := generateChannelDescription(name, input.URL)
		channels = append(channels, ChannelConfig{
			Name:        name,
			URL:         input.URL,
			Description: description,
		})
	}

	// Sort channels by name for consistent output
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Name < channels[j].Name
	})

	return channels, nil
}

// generateChannelDescription creates a user-friendly description for a channel
func generateChannelDescription(name, url string) string {
	switch {
	case strings.Contains(url, "nixpkgs-unstable"):
		return "Latest unstable packages"
	case strings.Contains(url, "nixos-"):
		// Extract version from URL like "github:NixOS/nixpkgs/nixos-24.05"
		parts := strings.Split(url, "/")
		if len(parts) >= 3 {
			version := strings.TrimPrefix(parts[len(parts)-1], "nixos-")
			return fmt.Sprintf("Stable packages (%s)", version)
		}
		return "Stable packages"
	case name == "nixpkgs":
		return "Default nixpkgs channel"
	case strings.Contains(url, "nixpkgs"):
		return "Custom nixpkgs channel"
	default:
		// For completely custom channels that don't contain "nixpkgs"
		return "Custom package channel"
	}
}

// AddChannel adds a new channel to devenv.yaml
func AddChannel(name, url string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")
	
	// Check if devenv.yaml exists
	if _, err := os.Stat(devenvYamlPath); os.IsNotExist(err) {
		return fmt.Errorf("devenv.yaml not found")
	}

	// Read and parse devenv.yaml
	content, err := os.ReadFile(devenvYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.yaml: %w", err)
	}

	var devenvConfig DevenvYaml
	if err := yaml.Unmarshal(content, &devenvConfig); err != nil {
		return fmt.Errorf("failed to parse devenv.yaml: %w", err)
	}

	// Initialize inputs map if it doesn't exist
	if devenvConfig.Inputs == nil {
		devenvConfig.Inputs = make(map[string]DevenvYamlInputs)
	}

	// Add the new channel
	devenvConfig.Inputs[name] = DevenvYamlInputs{
		URL: url,
	}

	// Marshal back to YAML
	updatedContent, err := yaml.Marshal(&devenvConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal devenv.yaml: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(devenvYamlPath, updatedContent, 0644); err != nil {
		return fmt.Errorf("failed to write devenv.yaml: %w", err)
	}

	return nil
}

// RemoveChannel removes a channel from devenv.yaml
func RemoveChannel(channelName string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")
	
	// Read current devenv.yaml
	content, err := os.ReadFile(devenvYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.yaml: %w", err)
	}

	// Parse YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("failed to parse devenv.yaml: %w", err)
	}

	// Check if inputs section exists
	inputs, ok := config["inputs"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no inputs section found in devenv.yaml")
	}

	// Check if channel exists
	if _, exists := inputs[channelName]; !exists {
		return fmt.Errorf("channel '%s' not found in devenv.yaml", channelName)
	}

	// Remove the channel
	delete(inputs, channelName)

	// Marshal back to YAML
	updatedContent, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated devenv.yaml: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(devenvYamlPath, updatedContent, 0644); err != nil {
		return fmt.Errorf("failed to write devenv.yaml: %w", err)
	}

	return nil
}

// GetChannelUsage returns which packages are using each channel
func GetChannelUsage() (map[string][]string, error) {
	// This is a placeholder for future implementation
	// Would need to parse devenv.nix and track which packages use which channels
	return map[string][]string{}, nil
}
