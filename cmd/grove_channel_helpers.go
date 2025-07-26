package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
)

// GitHubCommitInfo represents commit information from GitHub API
type GitHubCommitInfo struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Date string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
}

// isProtectedChannel checks if a channel is protected from removal
func isProtectedChannel(channelName string) bool {
	protectedChannels := map[string]bool{
		"nixpkgs":        true,
		"nixpkgs-stable": true,
	}
	return protectedChannels[channelName]
}

// getPackagesUsingChannel returns a list of packages that are using the specified channel
func getPackagesUsingChannel(channelName string) ([]string, error) {
	// Get the expected package prefix for this channel
	var packagePrefix string
	if channelName == "nixpkgs" {
		packagePrefix = "pkgs."
	} else {
		// Convert channel name to package prefix (e.g., "custom-elm" -> "pkgs-custom_elm.")
		packagePrefix = "pkgs-" + strings.ReplaceAll(channelName, "-", "_") + "."
	}

	// Get all Kanuka-managed packages (if devenv.nix exists)
	packages, err := grove.GetKanukaManagedPackages()
	if err != nil {
		// If devenv.nix doesn't exist, no packages are using any channels
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get managed packages: %w", err)
	}

	var usingChannel []string
	for _, pkg := range packages {
		if strings.HasPrefix(pkg, packagePrefix) {
			// Extract package name from full nix name (e.g., "pkgs-custom_elm.elm" -> "elm")
			parts := strings.Split(pkg, ".")
			if len(parts) >= 2 {
				packageName := parts[len(parts)-1]
				usingChannel = append(usingChannel, packageName)
			}
		}
	}

	return usingChannel, nil
}

// checkURLAccessibility checks if a channel URL is accessible
func checkURLAccessibility(url string) string {
	// For now, just return a basic status
	// TODO: Implement actual URL/Git accessibility checking
	if strings.Contains(url, "github.com") || strings.Contains(url, "github:") {
		return color.GreenString("✓") + " URL format valid"
	}
	return color.YellowString("?") + " Custom URL (not validated)"
}

// getOfficialChannelMetadata attempts to get additional metadata for official nixpkgs channels
func getOfficialChannelMetadata(url string) (commitInfo, lastUpdated, status string) {
	// Check if this is a GitHub nixpkgs URL
	if !strings.Contains(url, "github:NixOS/nixpkgs") {
		return "", "", checkURLAccessibility(url)
	}

	// Extract branch/ref from URL (e.g., "github:NixOS/nixpkgs/nixos-24.05" -> "nixos-24.05")
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return "", "", checkURLAccessibility(url)
	}

	ref := parts[len(parts)-1]
	if ref == "" {
		ref = "master" // fallback
	}

	// Fetch commit information from GitHub API
	commitInfo, lastUpdated = fetchGitHubCommitInfo("NixOS", "nixpkgs", ref)

	if commitInfo != "" {
		status = color.GreenString("✓") + " Accessible"
	} else {
		status = color.YellowString("?") + " Could not fetch metadata"
	}

	return commitInfo, lastUpdated, status
}

// fetchGitHubCommitInfo fetches the latest commit information from GitHub API
func fetchGitHubCommitInfo(owner, repo, ref string) (commitInfo, lastUpdated string) {
	// GitHub API URL for getting the latest commit on a branch/ref
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, ref)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the request
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", ""
	}

	// Parse the JSON response
	var commit GitHubCommitInfo
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", ""
	}

	// Format commit info
	shortSHA := commit.SHA
	if len(shortSHA) > 8 {
		shortSHA = shortSHA[:8]
	}

	// Get first line of commit message
	message := commit.Commit.Message
	if idx := strings.Index(message, "\n"); idx != -1 {
		message = message[:idx]
	}
	if len(message) > 50 {
		message = message[:47] + "..."
	}

	commitInfo = fmt.Sprintf("%s (%s)", shortSHA, message)

	// Parse and format the timestamp
	if parsedTime, err := time.Parse(time.RFC3339, commit.Commit.Author.Date); err == nil {
		lastUpdated = parsedTime.UTC().Format("2006-01-02 15:04:05 UTC")
	}

	return commitInfo, lastUpdated
}

// isPinnedChannel checks if a channel is a pinned channel based on naming pattern
func isPinnedChannel(channelName string) bool {
	return strings.Contains(channelName, "-pinned-")
}

// getPinnedChannelAge calculates the age of a pinned channel by fetching commit date
func getPinnedChannelAge(channelName, url string) (time.Duration, error) {
	// Extract commit hash from pinned channel name
	parts := strings.Split(channelName, "-pinned-")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid pinned channel name format")
	}

	shortHash := parts[1]

	// Fetch commit info to get the date
	_, lastUpdated := fetchGitHubCommitInfo("NixOS", "nixpkgs", shortHash)
	if lastUpdated == "" {
		return 0, fmt.Errorf("could not fetch commit date")
	}

	// Parse the timestamp
	commitTime, err := time.Parse("2006-01-02 15:04:05 UTC", lastUpdated)
	if err != nil {
		return 0, fmt.Errorf("could not parse commit date: %w", err)
	}

	return time.Since(commitTime), nil
}

// shouldWarnAboutPinnedChannel checks if a pinned channel is older than 6 months
func shouldWarnAboutPinnedChannel(channelName, url string) (bool, string) {
	if !isPinnedChannel(channelName) {
		return false, ""
	}

	age, err := getPinnedChannelAge(channelName, url)
	if err != nil {
		return false, ""
	}

	sixMonths := 6 * 30 * 24 * time.Hour // Approximate 6 months
	if age > sixMonths {
		months := int(age.Hours() / (24 * 30))
		return true, fmt.Sprintf("%d months old", months)
	}

	return false, ""
}

// checkUpdateNeeded determines if a channel needs updating and returns the new URL
func checkUpdateNeeded(channel grove.ChannelConfig, behavior UpdateBehavior) (bool, string, error) {
	switch behavior.ChannelType {
	case "official":
		return checkOfficialChannelUpdate(channel)
	case "pinned":
		return checkPinnedChannelUpdate(channel)
	default:
		return false, "", fmt.Errorf("cannot update custom channels")
	}
}

// checkOfficialChannelUpdate checks if an official nixpkgs channel needs updating
func checkOfficialChannelUpdate(channel grove.ChannelConfig) (bool, string, error) {
	// For unstable channel, it's always "latest" so no update needed
	if strings.Contains(channel.URL, "nixpkgs-unstable") {
		return false, "", nil
	}

	// For stable channels, check if there's a newer stable release
	if strings.Contains(channel.URL, "nixos-") {
		currentVersion := extractVersionFromURL(channel.URL)
		latestStable := grove.GetLatestStableChannel()

		if currentVersion != latestStable {
			newURL := "github:NixOS/nixpkgs/" + latestStable
			return true, newURL, nil
		}
	}

	return false, "", nil
}

// checkPinnedChannelUpdate checks if a pinned channel can be updated to latest commit
func checkPinnedChannelUpdate(channel grove.ChannelConfig) (bool, string, error) {
	// Extract current commit from URL
	parts := strings.Split(channel.URL, "/")
	if len(parts) < 3 {
		return false, "", fmt.Errorf("invalid pinned channel URL format")
	}

	currentCommit := parts[len(parts)-1]

	// Determine original branch from pinned channel name
	originalBranch := getOriginalBranchFromPinnedChannel(channel.Name)

	// Get latest commit from the original branch
	latestCommit, _ := fetchGitHubCommitInfo("NixOS", "nixpkgs", originalBranch)
	if latestCommit == "" {
		return false, "", fmt.Errorf("could not fetch latest commit for branch %s", originalBranch)
	}

	// Extract just the commit hash (first 8 chars for comparison)
	latestCommitHash := strings.Split(latestCommit, " ")[0]
	if len(latestCommitHash) > 8 {
		latestCommitHash = latestCommitHash[:8]
	}

	// Compare with current commit (first 8 chars)
	currentCommitShort := currentCommit
	if len(currentCommitShort) > 8 {
		currentCommitShort = currentCommitShort[:8]
	}

	if currentCommitShort != latestCommitHash {
		// Get the full commit hash for the new URL
		fullLatestCommit := strings.Split(latestCommit, " ")[0]
		newURL := "github:NixOS/nixpkgs/" + fullLatestCommit
		return true, newURL, nil
	}

	return false, "", nil
}

// getOriginalBranchFromPinnedChannel extracts the original branch from a pinned channel name
func getOriginalBranchFromPinnedChannel(pinnedChannelName string) string {
	// Extract base channel name from pinned name
	// e.g., "nixpkgs-pinned-abc123" -> "nixpkgs"
	parts := strings.Split(pinnedChannelName, "-pinned-")
	if len(parts) != 2 {
		return "nixpkgs-unstable" // fallback
	}

	baseChannel := parts[0]
	switch baseChannel {
	case "nixpkgs":
		return "nixpkgs-unstable"
	case "nixpkgs-stable":
		return grove.GetLatestStableChannel()
	default:
		return "nixpkgs-unstable" // fallback
	}
}

// extractVersionFromURL extracts version information from a channel URL for display
func extractVersionFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return "unknown"
	}

	lastPart := parts[len(parts)-1]

	// Handle different URL formats
	if strings.HasPrefix(lastPart, "nixos-") {
		return lastPart // e.g., "nixos-24.05"
	}
	if lastPart == "nixpkgs-unstable" {
		return "unstable"
	}
	if len(lastPart) >= 8 && isCommitHash(lastPart) {
		return lastPart[:8] + "..." // e.g., "abc123de..."
	}

	return lastPart
}

// isCommitHash checks if a string looks like a Git commit hash
func isCommitHash(s string) bool {
	if len(s) < 8 || len(s) > 40 {
		return false
	}
	for _, char := range s {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

// isOfficialNixpkgsChannel checks if a channel URL is an official nixpkgs channel
func isOfficialNixpkgsChannel(url string) bool {
	return strings.Contains(url, "github:NixOS/nixpkgs") || strings.Contains(url, "github.com/NixOS/nixpkgs")
}
