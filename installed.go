package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
)

type InstalledEntry struct {
	Repo           string                 `toml:"repo"`
	Target         string                 `toml:"target"`
	InstalledAt    time.Time              `toml:"installed_at"`
	URL            string                 `toml:"url"`
	Asset          string                 `toml:"asset"`
	Tool           string                 `toml:"tool,omitempty"`
	ExtractedFiles []string               `toml:"extracted_files"`
	Options        map[string]interface{} `toml:"options"`
	Version        string                 `toml:"version,omitempty"`
	Tag            string                 `toml:"tag,omitempty"`
	ReleaseDate    time.Time              `toml:"release_date,omitempty"`
}

type InstalledConfig struct {
	Installed map[string]InstalledEntry `toml:"installed"`
}

type UpgradeCandidate struct {
	Repo  string
	Entry InstalledEntry
}

type UpgradeResult struct {
	Repo    string
	Current string
	Latest  string
	Action  string // "upgrade", "skip", "error"
	Error   string
}

// getInstalledConfigPath returns the path to the installed packages config file
func getInstalledConfigPath() string {
	homePath, _ := os.UserHomeDir()

	// Use the same logic as existing config but for installed.toml
	configPath := filepath.Join(homePath, ".eget.installed.toml")

	// Check if it exists, if not try the XDG config directory
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		var configDir string
		switch runtime.GOOS {
		case "windows":
			configDir = os.Getenv("LOCALAPPDATA")
		default:
			configDir = os.Getenv("XDG_CONFIG_HOME")
		}
		if configDir == "" {
			configDir = filepath.Join(homePath, ".config")
		}
		xdgPath := filepath.Join(configDir, "eget", "installed.toml")
		return xdgPath
	}

	return configPath
}

// loadInstalledConfig loads the installed packages config from file
func loadInstalledConfig() (*InstalledConfig, error) {
	configPath := getInstalledConfigPath()

	var config InstalledConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load installed config: %w", err)
	}

	if config.Installed == nil {
		config.Installed = make(map[string]InstalledEntry)
	}

	return &config, nil
}

// saveInstalledConfig saves the installed packages config to file
func saveInstalledConfig(config *InstalledConfig) error {
	configPath := getInstalledConfigPath()

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// normalizeRepoName converts various target formats to a consistent repo key
func normalizeRepoName(target string) string {
	// Handle GitHub URLs
	if strings.Contains(target, "github.com/") {
		// Extract user/repo from github.com/user/repo or full URLs
		parts := strings.Split(target, "github.com/")
		if len(parts) > 1 {
			path := parts[1]
			// Remove trailing slashes and .git
			path = strings.TrimSuffix(path, "/")
			path = strings.TrimSuffix(path, ".git")
			// Take only user/repo part
			if idx := strings.Index(path, "/"); idx > 0 {
				repoPart := path[:idx+1+strings.Index(path[idx+1:], "/")]
				if repoPart == "" {
					repoPart = path
				}
				return strings.TrimSuffix(repoPart, "/")
			}
			return path
		}
	}

	// For direct repo names like "user/repo"
	if strings.Count(target, "/") == 1 && !strings.Contains(target, "://") {
		return target
	}

	// For other URLs or local paths, use as-is but clean up
	return strings.TrimSuffix(target, "/")
}

// extractOptionsMap converts Flags struct to a map for TOML storage
func extractOptionsMap(opts Flags) map[string]interface{} {
	options := make(map[string]interface{})

	// Only store meaningful options that affect installation
	if opts.Tag != "" {
		options["tag"] = opts.Tag
	}
	if opts.System != "" {
		options["system"] = opts.System
	}
	if opts.Output != "" {
		options["output"] = opts.Output
	}
	if opts.ExtractFile != "" {
		options["extract_file"] = opts.ExtractFile
	}
	if opts.All {
		options["all"] = opts.All
	}
	if opts.Quiet {
		options["quiet"] = opts.Quiet
	}
	if opts.DLOnly {
		options["download_only"] = opts.DLOnly
	}
	if opts.UpgradeOnly {
		options["upgrade_only"] = opts.UpgradeOnly
	}
	if len(opts.Asset) > 0 {
		options["asset"] = opts.Asset
	}
	if opts.Hash {
		options["hash"] = opts.Hash
	}
	if opts.Verify != "" {
		options["verify"] = opts.Verify
	}
	if opts.DisableSSL {
		options["disable_ssl"] = opts.DisableSSL
	}

	return options
}

// getReleaseInfo fetches tag and release date from GitHub API
func getReleaseInfo(repo, url string) (string, time.Time, error) {
	// Extract repo from URL if it's a GitHub URL
	var apiURL string
	if strings.Contains(url, "github.com/") && strings.Contains(url, "/releases/download/") {
		// Parse GitHub release URL to get repo and tag
		parts := strings.Split(url, "github.com/")
		if len(parts) < 2 {
			return "", time.Time{}, fmt.Errorf("invalid GitHub URL")
		}
		pathParts := strings.Split(parts[1], "/")
		if len(pathParts) < 4 {
			return "", time.Time{}, fmt.Errorf("invalid GitHub release URL")
		}
		repoName := pathParts[0] + "/" + pathParts[1]
		tag := pathParts[3] // tag is in position 3 for /releases/download/tag/asset

		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repoName, tag)
	} else if strings.Contains(repo, "/") {
		// For direct repo names, we need to find which tag was used
		// This is more complex, so for now we'll leave it empty
		return "", time.Time{}, nil
	} else {
		return "", time.Time{}, nil
	}

	// Make API request
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, nil // Don't fail if we can't get release info
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}

	var release struct {
		TagName   string    `json:"tag_name"`
		CreatedAt time.Time `json:"created_at"`
	}

	err = json.Unmarshal(body, &release)
	if err != nil {
		return "", time.Time{}, err
	}

	return release.TagName, release.CreatedAt, nil
}

// recordInstallation records a successful installation
func recordInstallation(target, url, tool string, opts Flags, extractedFiles []string) error {
	config, err := loadInstalledConfig()
	if err != nil {
		return err
	}

	repoKey := normalizeRepoName(target)

	// Try to get release information
	tag, releaseDate, _ := getReleaseInfo(repoKey, url)

	entry := InstalledEntry{
		Repo:           repoKey,
		Target:         target,
		InstalledAt:    time.Now(),
		URL:            url,
		Asset:          filepath.Base(url),
		Tool:           tool,
		ExtractedFiles: extractedFiles,
		Options:        extractOptionsMap(opts),
		Tag:            tag,
		ReleaseDate:    releaseDate,
	}

	// Store entry
	if config.Installed == nil {
		config.Installed = make(map[string]InstalledEntry)
	}
	config.Installed[repoKey] = entry

	return saveInstalledConfig(config)
}

// removeInstalled removes an installed package from tracking
func removeInstalled(target string) error {
	config, err := loadInstalledConfig()
	if err != nil {
		return err
	}

	repoKey := normalizeRepoName(target)
	delete(config.Installed, repoKey)

	return saveInstalledConfig(config)
}

// listInstalled displays all installed packages
func listInstalled() error {
	config, err := loadInstalledConfig()
	if err != nil {
		return err
	}

	if len(config.Installed) == 0 {
		fmt.Println("No packages installed.")
		return nil
	}

	fmt.Println("Installed packages:")
	fmt.Println()

	for _, entry := range config.Installed {
		fmt.Printf("%s\n", entry.Repo)
		fmt.Printf("  Target: %s\n", entry.Target)
		fmt.Printf("  Installed: %s\n", entry.InstalledAt.Format("2006-01-02 15:04:05"))
		if len(entry.ExtractedFiles) == 1 {
			fmt.Printf("  File: %s\n", entry.ExtractedFiles[0])
		} else {
			fmt.Printf("  Files: %s\n", strings.Join(entry.ExtractedFiles, ", "))
		}

		if len(entry.Options) > 0 {
			var opts []string
			for k, v := range entry.Options {
				opts = append(opts, fmt.Sprintf("%s=%v", k, v))
			}
			fmt.Printf("  Options: %s\n", strings.Join(opts, ", "))
		}
		fmt.Println()
	}

	return nil
}

// checkForUpgrade checks if a package has a newer version available
func checkForUpgrade(entry InstalledEntry) (bool, string, error) {
	// For GitHub repos, check if there's a newer release
	if !strings.Contains(entry.Repo, "/") {
		return false, "", fmt.Errorf("non-GitHub repos not supported for upgrade checks")
	}

	// Create a GithubAssetFinder to check for newer releases
	finder := &GithubAssetFinder{
		Repo:       entry.Repo,
		Tag:        "latest",
		Prerelease: false, // Only check stable releases
		MinTime:    entry.ReleaseDate,
	}

	// If we find assets, it means there's a newer release
	_, err := finder.Find()
	if err == ErrNoUpgrade {
		// No upgrade available
		return false, entry.Tag, nil
	} else if err != nil {
		return false, "", err
	}

	// There are assets, so there's an upgrade available
	// Get the latest tag
	latestTag, err := getLatestTag(entry.Repo)
	if err != nil {
		return false, "", err
	}

	return true, latestTag, nil
}

// getLatestTag gets the latest stable release tag for a repo
func getLatestTag(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}

	err = json.Unmarshal(body, &release)
	if err != nil {
		return "", err
	}

	if release.Prerelease {
		// If latest is a pre-release, we might need to find the latest stable
		// For simplicity, we'll accept it for now
	}

	return release.TagName, nil
}

// performUpgrade performs an upgrade for a single package
func performUpgrade(entry InstalledEntry, newTag string) error {
	// This is complex - we need to recreate the installation process
	// For now, we'll use a simplified approach by calling eget recursively
	// In a full implementation, this would parse the stored options and recreate the installation

	// Extract options back to command line args
	args := []string{entry.Target}

	// Add stored options
	opts := entry.Options
	if tag, ok := opts["tag"].(string); ok && tag != "" {
		args = append(args, "--tag", newTag) // Use new tag instead of old
	} else {
		args = append(args, "--tag", newTag) // Force the new tag
	}

	if system, ok := opts["system"].(string); ok && system != "" {
		args = append(args, "--system", system)
	}

	if extractFile, ok := opts["extract_file"].(string); ok && extractFile != "" {
		args = append(args, "--file", extractFile)
	}

	if all, ok := opts["all"].(bool); ok && all {
		args = append(args, "--all")
	}

	if quiet, ok := opts["quiet"].(bool); ok && quiet {
		args = append(args, "--quiet")
	}

	// Get the path to eget binary
	egetPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Run eget with the constructed arguments
	cmd := exec.Command(egetPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// updateInstalledRecord updates the installed record after successful upgrade
func updateInstalledRecord(repo, newTag string) error {
	config, err := loadInstalledConfig()
	if err != nil {
		return err
	}

	if entry, exists := config.Installed[repo]; exists {
		entry.Tag = newTag
		entry.InstalledAt = time.Now()
		// Note: We could fetch the new release date here, but for simplicity
		// we'll leave it as-is since the upgrade succeeded
		config.Installed[repo] = entry

		return saveInstalledConfig(config)
	}

	return fmt.Errorf("package %s not found in installed records", repo)
}

// upgradeAllPackages checks and upgrades all installed packages
func upgradeAllPackages(dryRun, interactive bool) ([]UpgradeResult, error) {
	config, err := loadInstalledConfig()
	if err != nil {
		return nil, err
	}

	// Convert installed entries to candidates
	candidates := make([]UpgradeCandidate, 0, len(config.Installed))
	for repo, entry := range config.Installed {
		candidates = append(candidates, UpgradeCandidate{
			Repo:  repo,
			Entry: entry,
		})
	}

	// If interactive mode, let user select which packages to check
	if interactive && len(candidates) > 0 {
		candidates = selectPackagesInteractively(candidates)
	}

	results := make([]UpgradeResult, 0, len(candidates))

	for _, candidate := range candidates {
		result := UpgradeResult{Repo: candidate.Repo, Current: candidate.Entry.Tag}

		needsUpgrade, latestTag, err := checkForUpgrade(candidate.Entry)
		if err != nil {
			result.Action = "error"
			result.Error = err.Error()
		} else if !needsUpgrade {
			result.Action = "skip"
			result.Latest = latestTag
		} else {
			result.Action = "upgrade"
			result.Latest = latestTag

			if !dryRun {
				err := performUpgrade(candidate.Entry, latestTag)
				if err != nil {
					result.Action = "error"
					result.Error = err.Error()
				} else {
					// Update the installed record
					updateErr := updateInstalledRecord(candidate.Repo, latestTag)
					if updateErr != nil {
						// Log but don't fail the upgrade
						result.Error = fmt.Sprintf("upgrade succeeded but record update failed: %v", updateErr)
					}
				}
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// Bubbletea model for interactive package selection
type packageSelectModel struct {
	candidates []UpgradeCandidate
	cursor     int
	selected   map[int]bool
	done       bool
}

func (m packageSelectModel) Init() tea.Cmd {
	return nil
}

func (m packageSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.done = true
			return m, tea.Quit
		case "enter", " ":
			// Toggle selection
			if _, exists := m.selected[m.cursor]; exists {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = true
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.candidates)-1 {
				m.cursor++
			}
		case "a", "ctrl+a":
			// Select all
			for i := range m.candidates {
				m.selected[i] = true
			}
		case "n", "ctrl+n":
			// Select none
			m.selected = make(map[int]bool)
		case "ctrl+d":
			// Done
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m packageSelectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString("Select packages to check for updates:\n\n")

	for i, candidate := range m.candidates {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = "[✓]"
		}

		current := candidate.Entry.Tag
		if current == "" {
			current = "unknown"
		}

		b.WriteString(fmt.Sprintf("%s %s %s (current: %s)\n", cursor, checkbox, candidate.Repo, current))
	}

	b.WriteString("\n")
	b.WriteString("↑/↓ or j/k: navigate • space: toggle • a: select all • n: select none\n")
	b.WriteString("ctrl+d or enter: confirm • q/esc: quit\n")

	return b.String()
}

// selectPackagesInteractively allows users to select which packages to upgrade using bubbletea
func selectPackagesInteractively(candidates []UpgradeCandidate) []UpgradeCandidate {
	if len(candidates) == 0 {
		return candidates
	}

	// Check if we're in an interactive terminal
	if !isInteractiveTerminal() {
		fmt.Fprintf(os.Stderr, "Warning: not running in interactive terminal, proceeding with all packages\n")
		return candidates
	}

	// Create and run the bubbletea program
	model := packageSelectModel{
		candidates: candidates,
		selected:   make(map[int]bool),
		done:       false,
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: interactive selection failed (%v), proceeding with all packages\n", err)
		return candidates
	}

	m := finalModel.(packageSelectModel)

	// Collect selected candidates
	selected := make([]UpgradeCandidate, 0, len(m.selected))
	for idx := range m.selected {
		selected = append(selected, candidates[idx])
	}

	return selected
}

// isInteractiveTerminal checks if we're running in an interactive terminal
func isInteractiveTerminal() bool {
	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// displayUpgradeResults shows the results of the upgrade-all operation
func displayUpgradeResults(results []UpgradeResult, dryRun bool) {
	if dryRun {
		fmt.Println("Dry run - the following packages would be upgraded:")
	} else {
		fmt.Println("Upgrade results:")
	}
	fmt.Println()

	upgraded := 0
	skipped := 0
	errors := 0

	for _, result := range results {
		switch result.Action {
		case "upgrade":
			if dryRun {
				fmt.Printf("✓ %s: %s → %s (would upgrade)\n", result.Repo, result.Current, result.Latest)
			} else {
				fmt.Printf("✓ %s: %s → %s (upgraded)\n", result.Repo, result.Current, result.Latest)
			}
			upgraded++
		case "skip":
			fmt.Printf("• %s: %s (up to date)\n", result.Repo, result.Current)
			skipped++
		case "error":
			fmt.Printf("✗ %s: %s\n", result.Repo, result.Error)
			errors++
		}
	}

	fmt.Println()
	fmt.Printf("Summary: %d upgraded, %d skipped, %d errors\n", upgraded, skipped, errors)
}
