package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/google/go-github/github"
)

///////////////////////////
//     GENERAL UTILS     //
///////////////////////////

// Check if a command exists on the system
func commandExists(commandName string) bool {
	_, err := exec.LookPath(commandName)
	return err == nil
}

// Run a system command and get output
func runCommand(command string) string {
	cmd := exec.Command("sh", "-c", command)
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(stdout)
}

// Download a file and return as byte array
func download(url string) []byte {
	// Get request
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read body
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// Return bytes
	return b
}

///////////////////////////
//        CONFIG         //
///////////////////////////

// Structs to store contents of config.toml
type (
	config struct {
		TmpDir          string `toml:"tmp_dir"`
		InstallURL      string `toml:"custom_install_url"`
		HelpDescription string `toml:"help_description"`
		Sync            map[string]targetClass
		Installers      map[string]Installer
		Metadata        metadata
	}

	targetClass struct {
		Name      string
		MacOSOnly bool `toml:"macos_only"`
		Targets   []Target
	}

	Target struct {
		Description string
		RepoPath    string `toml:"repo_path"`
		LocalPath   string `toml:"local_path"`
	}

	Installer struct {
		HelpMessage string `toml:"help_message"`
		Description string
		Install     []map[string]string
		TmpInstall  []map[string]string `toml:"tmp_install"`
	}

	metadata struct {
		User     string
		Repo     string
		BaseURL  string
		GitPaths []string
	}
)

// Create map of repo paths and local paths for all sync targets
func (c *config) SyncTargets() map[string]string {
	targets := make(map[string]string)
	for _, s := range Config.Sync {
		for _, t := range s.Targets {
			targets[t.RepoPath] = t.LocalPath
		}
	}
	return targets
}

// Get all paths in remote GitHub repo
func remoteGitPaths(user, repo, branch string) []string {
	// Get tree from GitHub API
	client := github.NewClient(nil)
	tree, _, err := client.Git.GetTree(context.Background(), user, repo, branch, true)
	if err != nil {
		log.Fatal(err)
	}

	// Get paths from tree
	var gitPaths []string
	for _, entry := range tree.Entries {
		gitPaths = append(gitPaths, entry.GetPath())
	}

	return gitPaths
}

// Unmarshall config.toml file and add metadata
func getConfig() config {
	var c config

	// Get git repo remote origin url and split into user and repo
	// We need to do this first to get the remote config file
	gitRemoteOriginURL := runCommand("git config --get remote.origin.url")
	splitURL := strings.Split(strings.TrimSpace(gitRemoteOriginURL), "/")
	c.Metadata.User = splitURL[len(splitURL)-2]
	c.Metadata.Repo = splitURL[len(splitURL)-1]

	// Get base URL for raw GitHub user content
	branch := strings.TrimSpace(runCommand("git rev-parse --abbrev-ref HEAD"))
	c.Metadata.BaseURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/", c.Metadata.User, c.Metadata.Repo, branch)

	// Read text of TOML file
	configToml := download(c.Metadata.BaseURL + "config.toml")

	// Unmarshall TOML file
	_, err := toml.Decode(string(configToml), &c)
	if err != nil {
		log.Fatal(err)
	}

	// Update TmpDir with repo info
	c.TmpDir = strings.ReplaceAll(c.TmpDir, "@repo_name", c.Metadata.Repo)

	// If no custom install url, set install url
	if c.InstallURL == "" {
		c.InstallURL = c.Metadata.BaseURL + "install.sh"
	}

	// Get all paths in remote GitHub repo
	c.Metadata.GitPaths = remoteGitPaths(c.Metadata.User, c.Metadata.Repo, branch)

	return c
}

// Make config data globally available
var Config config = getConfig()

///////////////////////////
//    PACKAGE MANAGER    //
///////////////////////////

// Store package manager commands and available packages
type (
	packageManager struct {
		commands pmCommands
		packages pkgs
	}

	pmCommands struct {
		name         string
		installCmd   string
		uninstallCmd string
		updateCmd    string
	}

	pkgs map[string]map[string]map[string]string
)

// Get a package by its name
func (packages *pkgs) packageByName(name string) map[string]string {
	for _, group := range *packages {
		for pName, p := range group {
			if pName == name {
				return p
			}
		}
	}
	return map[string]string{}
}

// Get system install command for a given package
func (pm *packageManager) installCmd(name string) string {
	// Get package from packages.toml
	pack := pm.packages.packageByName(name)

	// Check if pack has key "install_command" and return it if it does
	if installCmd, ok := pack["install_command"]; ok {
		return installCmd
	}

	// Check for system package name and return install command if it exists
	if systemPackageName, ok := pack[pm.commands.name]; ok {
		return pm.commands.installCmd + " " + systemPackageName
	}

	// Package not found, return empty string
	return ""
}

// Get system uninstall command for a given package
func (pm *packageManager) uninstallCmd(name string) string {
	// Get package from packages.toml
	pack := pm.packages.packageByName(name)

	// Check if pack has key "uninstall_command" and return it if it does
	if installCmd, ok := pack["uninstall_command"]; ok {
		return installCmd
	}

	// Check for system package name and return uninstall command if it exists
	if systemPackageName, ok := pack[pm.commands.name]; ok {
		return pm.commands.uninstallCmd + " " + systemPackageName
	}

	// Package not found, return empty string
	return ""
}

// Get system pacakge manager commands and listed packages
func getPackageManager() packageManager {
	var pm packageManager

	// Get package manager commands
	if commandExists("pacman") {
		pm.commands = pmCommands{
			name:         "pacman",
			installCmd:   "pacman -S --no-confirm",
			uninstallCmd: "pacman -Rs --no-confirm",
			updateCmd:    "pacman -Syu",
		}
	} else if commandExists("dnf") {
		pm.commands = pmCommands{
			name:         "dnf",
			installCmd:   "dnf install -y",
			uninstallCmd: "dnf remove -y",
			updateCmd:    "dnf update",
		}
	} else if commandExists("brew") {
		pm.commands = pmCommands{
			name:         "brew",
			installCmd:   "brew install",
			uninstallCmd: "brew uninstall",
			updateCmd:    "brew upgrade",
		}
	} else if commandExists("apt") {
		pm.commands = pmCommands{
			name:         "apt",
			installCmd:   "apt install -y",
			uninstallCmd: "apt remove -y",
			updateCmd:    "apt update",
		}
	}

	// Download packages TOML file from this repo
	tomlText := download(Config.Metadata.BaseURL + "packages.toml")

	// Unmarshal TOML file into struct
	_, err := toml.Decode(string(tomlText), &pm.packages)
	if err != nil {
		log.Fatal(err)
	}

	return pm
}

// Make package manager and packages available globally
var PM packageManager = getPackageManager()
