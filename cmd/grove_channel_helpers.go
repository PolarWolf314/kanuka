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