package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Config holds saved instance configurations
type Config struct {
	ChromeProfileDir string     `json:"chrome_profile_dir"`
	Instances        []Instance `json:"instances"`
}

// Instance holds GCP instance details
type Instance struct {
	Alias          string `json:"alias"`
	Project        string `json:"project"`
	Zone           string `json:"zone"`
	Name           string `json:"name"`
	AuthUser       int    `json:"authuser"`
	GcloudAccount  string `json:"gcloud_account,omitempty"`
	ConnectionMode string `json:"connection_mode,omitempty"` // browser or terminal
}

func main() {
	configPath := getConfigPath()
	config := loadConfig(configPath)

	if len(os.Args) > 1 {
		handleArgs(os.Args[1:], config, configPath)
		return
	}

	interactiveMode(config, configPath)
}

//  Config helpers

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".gcp-ssh")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

func loadConfig(path string) *Config {
	config := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return config
	}
	json.Unmarshal(data, config)
	return config
}

func saveConfig(path string, config *Config) {
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(path, data, 0644)
}

// ─── CLI argument handling ───────────────────────────────────────────────────

func handleArgs(args []string, config *Config, configPath string) {
	switch args[0] {
	case "list":
		listInstances(config)
	case "add":
		addInstance(config, configPath)
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: gcp-ssh remove <alias>")
			return
		}
		removeInstance(config, configPath, args[1])
	case "connect":
		if len(args) < 2 {
			fmt.Println("Usage: gcp-ssh connect <alias>")
			return
		}
		connectByAlias(config, args[1], "")
	case "connect-terminal":
		if len(args) < 2 {
			fmt.Println("Usage: gcp-ssh connect-terminal <alias>")
			return
		}
		connectByAlias(config, args[1], "terminal")
	case "quick":
		if len(args) < 4 {
			fmt.Println("Usage: gcp-ssh quick <project> <zone> <instance-name> [gcloud-account-email]")
			return
		}
		inst := Instance{
			Project:  args[1],
			Zone:     args[2],
			Name:     args[3],
			AuthUser: 0,
		}
		if len(args) > 4 {
			inst.GcloudAccount = args[4]
		}
		openSSH(config, inst)
	case "quick-terminal":
		if len(args) < 4 {
			fmt.Println("Usage: gcp-ssh quick-terminal <project> <zone> <instance-name> [gcloud-account-email]")
			return
		}
		inst := Instance{Project: args[1], Zone: args[2], Name: args[3], AuthUser: 0, ConnectionMode: "terminal"}
		if len(args) > 4 {
			inst.GcloudAccount = args[4]
		}
		connectTerminal(inst)
	case "profile":
		setChromeProfile(config, configPath)
	case "help":
		printHelp()
	default:
		// Try treating first arg as an alias
		connectByAlias(config, args[0], "")
	}
}

// ─── Interactive mode ────────────────────────────────────────────────────────

func interactiveMode(config *Config, configPath string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║      GCP SSH-in-Browser Launcher         ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	// Check Chrome profile
	if config.ChromeProfileDir == "" {
		fmt.Println("⚠  No Chrome profile set. Let's configure it first.")
		setChromeProfile(config, configPath)
		fmt.Println()
	}

	for {
		fmt.Println("┌─ What would you like to do?")
		fmt.Println("│  1) Connect to a saved instance")
		fmt.Println("│  2) Quick connect (enter details now)")
		fmt.Println("│  3) Add a new saved instance")
		fmt.Println("│  4) List saved instances")
		fmt.Println("│  5) Remove a saved instance")
		fmt.Println("│  6) Change Chrome profile")
		fmt.Println("│  7) Exit")
		fmt.Print("└─ Choice: ")

		choice := readLine(reader)

		switch choice {
		case "1":
			if len(config.Instances) == 0 {
				fmt.Println("\n  No saved instances. Add one first.")
				continue
			}
			fmt.Println()
			listInstances(config)
			fmt.Print("  Enter alias or number: ")
			input := readLine(reader)
			// Try as number first
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(config.Instances) {
				openByMode(config, config.Instances[num-1])
			} else {
				connectByAlias(config, input, "")
			}
			fmt.Println()

		case "2":
			fmt.Println()
			inst := promptInstanceDetails(reader)
			openByMode(config, inst)
			fmt.Println()

			fmt.Print("  Save this instance for later? (y/n): ")
			if strings.ToLower(readLine(reader)) == "y" {
				fmt.Print("  Enter an alias: ")
				inst.Alias = readLine(reader)
				config.Instances = append(config.Instances, inst)
				saveConfig(configPath, config)
				fmt.Printf("  ✓ Saved as '%s'\n\n", inst.Alias)
			}

		case "3":
			fmt.Println()
			addInstance(config, configPath)
			fmt.Println()

		case "4":
			fmt.Println()
			listInstances(config)
			fmt.Println()

		case "5":
			fmt.Println()
			listInstances(config)
			fmt.Print("  Enter alias to remove: ")
			alias := readLine(reader)
			removeInstance(config, configPath, alias)
			fmt.Println()

		case "6":
			fmt.Println()
			setChromeProfile(config, configPath)
			fmt.Println()

		case "7":
			fmt.Println("  Bye! 👋")
			return

		default:
			fmt.Println("  Invalid choice.")
		}
	}
}

// ─── Instance operations ─────────────────────────────────────────────────────

func promptInstanceDetails(reader *bufio.Reader) Instance {
	inst := Instance{}
	fmt.Print("  GCP Project ID: ")
	inst.Project = readLine(reader)
	fmt.Print("  Zone (e.g. us-central1-a): ")
	inst.Zone = readLine(reader)
	fmt.Print("  Instance Name: ")
	inst.Name = readLine(reader)
	fmt.Print("  Auth User index for browser URL (0 default, 1 second account, etc.) [0]: ")
	authStr := readLine(reader)
	if authStr == "" {
		inst.AuthUser = 0
	} else {
		inst.AuthUser, _ = strconv.Atoi(authStr)
	}
	fmt.Print("  Google account email for gcloud (recommended): ")
	inst.GcloudAccount = readLine(reader)
	fmt.Print("  Preferred SSH mode [browser/terminal] (default browser): ")
	mode := strings.ToLower(readLine(reader))
	if mode == "terminal" {
		inst.ConnectionMode = "terminal"
	} else {
		inst.ConnectionMode = "browser"
	}
	return inst
}

func addInstance(config *Config, configPath string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("  Enter an alias (short name): ")
	alias := readLine(reader)

	// Check for duplicate alias
	for _, inst := range config.Instances {
		if inst.Alias == alias {
			fmt.Printf("  ✗ Alias '%s' already exists.\n", alias)
			return
		}
	}

	inst := promptInstanceDetails(reader)
	inst.Alias = alias
	config.Instances = append(config.Instances, inst)
	saveConfig(configPath, config)
	fmt.Printf("  ✓ Instance '%s' saved.\n", alias)
}

func removeInstance(config *Config, configPath string, alias string) {
	for i, inst := range config.Instances {
		if inst.Alias == alias {
			config.Instances = append(config.Instances[:i], config.Instances[i+1:]...)
			saveConfig(configPath, config)
			fmt.Printf("  ✓ Removed '%s'.\n", alias)
			return
		}
	}
	fmt.Printf("  ✗ Alias '%s' not found.\n", alias)
}

func listInstances(config *Config) {
	if len(config.Instances) == 0 {
		fmt.Println("  No saved instances.")
		return
	}
	fmt.Println("  ┌─ Saved Instances:")
	for i, inst := range config.Instances {
		mode := inst.ConnectionMode
		if mode == "" {
			mode = "browser"
		}
		account := inst.GcloudAccount
		if account == "" {
			account = "(active gcloud account)"
		}
		fmt.Printf("  │  %d) [%s] %s/%s/%s (authuser=%d, mode=%s, account=%s)\n",
			i+1, inst.Alias, inst.Project, inst.Zone, inst.Name, inst.AuthUser, mode, account)
	}
	fmt.Println("  └─")
}

func connectByAlias(config *Config, alias string, forcedMode string) {
	for _, inst := range config.Instances {
		if inst.Alias == alias {
			if forcedMode != "" {
				inst.ConnectionMode = forcedMode
			}
			openByMode(config, inst)
			return
		}
	}
	fmt.Printf("  ✗ Alias '%s' not found. Use 'list' to see saved instances.\n", alias)
}

func openByMode(config *Config, inst Instance) {
	mode := inst.ConnectionMode
	if mode == "" {
		mode = "browser"
	}
	if mode == "terminal" {
		connectTerminal(inst)
		return
	}
	openSSH(config, inst)
}

// ─── Chrome profile ──────────────────────────────────────────────────────────

func setChromeProfile(config *Config, configPath string) {
	reader := bufio.NewReader(os.Stdin)

	profiles := discoverChromeProfiles()
	if len(profiles) > 0 {
		fmt.Println("  Discovered Chrome profiles:")
		for i, p := range profiles {
			fmt.Printf("    %d) %s\n", i+1, p)
		}
		fmt.Print("  Select a number, or type a custom profile directory name: ")
		input := readLine(reader)
		if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(profiles) {
			config.ChromeProfileDir = profiles[num-1]
		} else {
			config.ChromeProfileDir = input
		}
	} else {
		fmt.Println("  Could not auto-discover profiles.")
		fmt.Println("  Common values: 'Default', 'Profile 1', 'Profile 2', etc.")
		fmt.Print("  Enter Chrome profile directory name: ")
		config.ChromeProfileDir = readLine(reader)
	}

	saveConfig(configPath, config)
	fmt.Printf("  ✓ Chrome profile set to: %s\n", config.ChromeProfileDir)
}

func discoverChromeProfiles() []string {
	chromeUserDataDir := getChromeUserDataDir()
	if chromeUserDataDir == "" {
		return nil
	}

	var profiles []string
	entries, err := os.ReadDir(chromeUserDataDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Chrome profiles are "Default", "Profile 1", "Profile 2", etc.
		if name == "Default" || strings.HasPrefix(name, "Profile ") {
			// Try to read the profile name from Preferences
			displayName := getProfileDisplayName(filepath.Join(chromeUserDataDir, name))
			if displayName != "" {
				profiles = append(profiles, name+" ("+displayName+")")
			} else {
				profiles = append(profiles, name)
			}
		}
	}
	return profiles
}

func getProfileDisplayName(profilePath string) string {
	prefsPath := filepath.Join(profilePath, "Preferences")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return ""
	}
	var prefs map[string]any
	if err := json.Unmarshal(data, &prefs); err != nil {
		return ""
	}
	if profile, ok := prefs["profile"].(map[string]any); ok {
		if name, ok := profile["name"].(string); ok {
			return name
		}
	}
	return ""
}

// ─── SSH URL & launch ────────────────────────────────────────────────────────

func buildSSHURL(inst Instance) string {
	return fmt.Sprintf(
		"https://ssh.cloud.google.com/projects/%s/zones/%s/instances/%s?authuser=%d&hl=en_US&projectNumber=0",
		inst.Project, inst.Zone, inst.Name, inst.AuthUser,
	)
}

func openSSH(config *Config, inst Instance) {
	if !ensureInstanceReady(inst) {
		fmt.Println("  ⚠ Instance readiness could not be verified with gcloud. Please start it manually if needed.")
	}

	url := buildSSHURL(inst)
	chromePath := getChromeExecutable()

	if chromePath == "" {
		fmt.Println("  ✗ Could not find Chrome. Opening URL in default browser...")
		openURLDefault(url)
		return
	}

	profileDir := config.ChromeProfileDir
	if profileDir == "" {
		profileDir = "Default"
	}

	// Strip display name suffix if present, e.g. "Profile 1 (Work)" -> "Profile 1"
	if idx := strings.Index(profileDir, " ("); idx != -1 {
		profileDir = profileDir[:idx]
	}

	args := []string{"--profile-directory=" + profileDir, url}

	fmt.Printf("  🚀 Opening browser SSH for: %s (zone: %s, project: %s)\n", inst.Name, inst.Zone, inst.Project)
	fmt.Printf("  🌐 URL: %s\n", url)
	fmt.Printf("  🔑 Chrome profile: %s\n", profileDir)

	cmd := exec.Command(chromePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("  ✗ Failed to launch Chrome: %v\n", err)
		fmt.Println("  Trying default browser...")
		openURLDefault(url)
		return
	}

	fmt.Println("  ✓ Chrome launched! SSH session will authenticate automatically.")
}

func connectTerminal(inst Instance) {
	if !ensureInstanceReady(inst) {
		fmt.Println("  ✗ Cannot continue with terminal SSH until gcloud is available and the instance is running.")
		return
	}

	fmt.Printf("  🚀 Opening terminal SSH for: %s (zone: %s, project: %s)\n", inst.Name, inst.Zone, inst.Project)
	cmd := exec.Command("gcloud", "compute", "ssh", inst.Name, "--project", inst.Project, "--zone", inst.Zone)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("  ✗ gcloud compute ssh failed: %v\n", err)
	}
}

func ensureInstanceReady(inst Instance) bool {
	if _, err := exec.LookPath("gcloud"); err != nil {
		fmt.Println("  ⚠ gcloud CLI not found. Install gcloud or start the instance manually before SSH.")
		return false
	}

	if !ensureGcloudAccount(inst.GcloudAccount) {
		return false
	}

	if err := runGcloudCommand("config", "set", "project", inst.Project); err != nil {
		fmt.Printf("  ✗ Failed to set active gcloud project: %v\n", err)
		return false
	}

	status, err := runGcloudValueCommand("compute", "instances", "describe", inst.Name,
		"--project", inst.Project,
		"--zone", inst.Zone,
		"--format=value(status)")
	if err != nil {
		fmt.Printf("  ✗ Failed to read instance status: %v\n", err)
		return false
	}

	if strings.EqualFold(status, "RUNNING") {
		fmt.Println("  ✓ Instance is already running.")
		return true
	}

	fmt.Printf("  ℹ Instance status is '%s'. Starting instance...\n", status)
	if err := runGcloudCommand("compute", "instances", "start", inst.Name,
		"--project", inst.Project,
		"--zone", inst.Zone); err != nil {
		fmt.Printf("  ✗ Failed to start instance: %v\n", err)
		return false
	}

	fmt.Println("  ✓ Instance started.")
	return true
}

func ensureGcloudAccount(requiredAccount string) bool {
	active, err := runGcloudValueCommand("auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	if err != nil {
		fmt.Printf("  ✗ Failed to determine active gcloud account: %v\n", err)
		return false
	}

	if requiredAccount == "" {
		if active == "" {
			fmt.Println("  ⚠ No active gcloud account. Please run 'gcloud auth login' and retry.")
			return false
		}
		fmt.Printf("  ✓ Using active gcloud account: %s\n", active)
		return true
	}

	if active == requiredAccount {
		fmt.Printf("  ✓ gcloud account matches configured account: %s\n", requiredAccount)
		return true
	}

	fmt.Printf("  ℹ Active gcloud account is '%s', but '%s' is required.\n", active, requiredAccount)
	accounts, err := runGcloudValueCommand("auth", "list", "--format=value(account)")
	if err == nil {
		for _, account := range strings.Split(accounts, "\n") {
			if strings.TrimSpace(account) == requiredAccount {
				fmt.Printf("  ℹ Switching gcloud account to '%s'...\n", requiredAccount)
				if err := runGcloudCommand("config", "set", "account", requiredAccount); err != nil {
					fmt.Printf("  ✗ Failed to set gcloud account: %v\n", err)
					return false
				}
				return true
			}
		}
	}

	fmt.Printf("  ℹ Logging in to gcloud as '%s'...\n", requiredAccount)
	if err := runGcloudCommand("auth", "login", requiredAccount); err != nil {
		fmt.Printf("  ✗ gcloud login failed for '%s': %v\n", requiredAccount, err)
		return false
	}
	if err := runGcloudCommand("config", "set", "account", requiredAccount); err != nil {
		fmt.Printf("  ✗ Failed to set gcloud account after login: %v\n", err)
		return false
	}
	return true
}

func runGcloudCommand(args ...string) error {
	cmd := exec.Command("gcloud", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGcloudValueCommand(args ...string) (string, error) {
	cmd := exec.Command("gcloud", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ─── Platform-specific helpers ───────────────────────────────────────────────

func getChromeExecutable() string {
	switch runtime.GOOS {
	case "darwin":
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "windows":
		paths := []string{
			filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "linux":
		names := []string{"google-chrome", "google-chrome-stable", "chromium-browser", "chromium"}
		for _, name := range names {
			if path, err := exec.LookPath(name); err == nil {
				return path
			}
		}
	}
	return ""
}

func getChromeUserDataDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
	case "linux":
		return filepath.Join(home, ".config", "google-chrome")
	}
	return ""
}

func openURLDefault(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

// ─── Utilities ───────────────────────────────────────────────────────────────

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func printHelp() {
	fmt.Println(`
GCP SSH Launcher (Browser + Terminal)
=====================================

Usage:
  gcp-ssh                                   Interactive mode
  gcp-ssh <alias>                           Connect to saved instance by alias
  gcp-ssh connect <alias>                   Connect to saved instance by alias
  gcp-ssh connect-terminal <alias>          Force terminal SSH mode for alias
  gcp-ssh quick <project> <zone> <vm> [acc] One-off quick connect in browser mode
  gcp-ssh quick-terminal <project> <zone> <vm> [acc]
                                            One-off quick connect in terminal mode
  gcp-ssh add                               Add a new saved instance
  gcp-ssh list                              List saved instances
  gcp-ssh remove <alias>                    Remove a saved instance
  gcp-ssh profile                           Change Chrome profile
  gcp-ssh help                              Show this help

Before SSH, the tool now:
  1) verifies gcloud CLI is installed,
  2) verifies/sets the expected gcloud account,
  3) checks and starts the instance if not running.

Config is stored at: ~/.gcp-ssh/config.json`)
}
