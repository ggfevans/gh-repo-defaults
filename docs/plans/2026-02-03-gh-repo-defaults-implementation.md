# gh-repo-defaults Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a `gh` CLI extension that creates GitHub repos with consistent defaults (labels, settings, boilerplate, branch protection) from named YAML profiles, with a Bubble Tea TUI.

**Architecture:** Go CLI extension using Cobra for commands, Bubble Tea + huh for TUI forms, shelling out to `gh` CLI for all GitHub API interactions. Config stored in `~/.config/gh-repo-defaults/config.yaml` with named profiles. Embedded default templates via `go:embed`.

**Tech Stack:** Go 1.22+, Cobra, Bubble Tea, huh, Lip Gloss, yaml.v3

**Design doc:** `docs/plans/2026-02-03-gh-repo-defaults-design.md`

---

### Task 1: Project Scaffolding

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `cmd/root.go`

**Step 1: Initialize Go module**

Run: `go mod init github.com/gvns/gh-repo-defaults`
Expected: `go.mod` created

**Step 2: Install dependencies**

Run:
```bash
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/huh@latest
go get gopkg.in/yaml.v3
```
Expected: dependencies added to `go.mod` and `go.sum`

**Step 3: Write main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/gvns/gh-repo-defaults/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 4: Write cmd/root.go**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-repo-defaults",
	Short: "Create GitHub repos with consistent defaults",
	Long:  "A gh CLI extension that creates GitHub repos and applies labels, settings, boilerplate, and branch protection from named profiles.",
}

func Execute() error {
	return rootCmd.Execute()
}
```

**Step 5: Verify it compiles**

Run: `go build -o gh-repo-defaults .`
Expected: binary created, runs with `./gh-repo-defaults --help`

**Step 6: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go
git commit -m "feat: scaffold Go project with Cobra root command"
```

---

### Task 2: Config Types & Loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the test for config loading**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	yaml := `default_profile: oss
default_owner: ""
profiles:
  oss:
    description: "Open source project defaults"
    settings:
      has_wiki: false
      has_projects: false
      delete_branch_on_merge: true
      allow_squash_merge: true
    labels:
      clear_existing: true
      items:
        - name: bug
          color: "d73a4a"
          description: "Something isn't working"
        - name: enhancement
          color: "a2eeef"
    boilerplate:
      license: MIT
      gitignore: Go
    branch_protection:
      branch: main
      required_reviews: 0
`
	if err := os.WriteFile(configPath, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if cfg.DefaultProfile != "oss" {
		t.Errorf("DefaultProfile = %q, want %q", cfg.DefaultProfile, "oss")
	}
	p, ok := cfg.Profiles["oss"]
	if !ok {
		t.Fatal("missing profile 'oss'")
	}
	if p.Settings.HasWiki {
		t.Error("HasWiki should be false")
	}
	if !p.Settings.AllowSquashMerge {
		t.Error("AllowSquashMerge should be true")
	}
	if len(p.Labels.Items) != 2 {
		t.Errorf("Labels.Items len = %d, want 2", len(p.Labels.Items))
	}
	if p.Labels.Items[0].Name != "bug" {
		t.Errorf("first label = %q, want %q", p.Labels.Items[0].Name, "bug")
	}
	if p.Boilerplate.License != "MIT" {
		t.Errorf("License = %q, want %q", p.Boilerplate.License, "MIT")
	}
	if p.BranchProtection.Branch != "main" {
		t.Errorf("Branch = %q, want %q", p.BranchProtection.Branch, "main")
	}
}

func TestLoadConfig_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	// Write a file larger than 1MB
	data := make([]byte, 1024*1024+1)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Fatal("expected error for oversized config")
	}
}

func TestLoadConfig_MissingFile_ReturnsDefault(t *testing.T) {
	cfg, err := LoadFromFile("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.DefaultProfile != "personal" {
		t.Errorf("DefaultProfile = %q, want %q", cfg.DefaultProfile, "personal")
	}
	if len(cfg.Profiles) == 0 {
		t.Error("expected built-in profiles")
	}
}

func TestLoadConfig_PermissionWarning(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	yaml := `default_profile: oss
profiles: {}
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFromFile(configPath)
	// Should succeed but log a warning — we test that it at least doesn't fail
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `LoadFromFile` not defined

**Step 3: Write the config types and loader**

```go
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

const maxConfigSize = 1024 * 1024 // 1MB

// Config is the top-level configuration.
type Config struct {
	DefaultProfile string             `yaml:"default_profile"`
	DefaultOwner   string             `yaml:"default_owner"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

// Profile defines defaults to apply to a repo.
type Profile struct {
	Description      string           `yaml:"description"`
	Settings         RepoSettings     `yaml:"settings"`
	Labels           LabelConfig      `yaml:"labels"`
	Boilerplate      BoilerplateConfig `yaml:"boilerplate"`
	BranchProtection BranchProtection `yaml:"branch_protection"`
}

// RepoSettings maps to GitHub repo API fields.
type RepoSettings struct {
	HasWiki                   bool   `yaml:"has_wiki"`
	HasProjects               bool   `yaml:"has_projects"`
	HasDiscussions            bool   `yaml:"has_discussions"`
	DeleteBranchOnMerge       bool   `yaml:"delete_branch_on_merge"`
	AllowSquashMerge          bool   `yaml:"allow_squash_merge"`
	AllowMergeCommit          bool   `yaml:"allow_merge_commit"`
	AllowRebaseMerge          bool   `yaml:"allow_rebase_merge"`
	SquashMergeCommitTitle    string `yaml:"squash_merge_commit_title"`
	SquashMergeCommitMessage  string `yaml:"squash_merge_commit_message"`
}

// LabelConfig defines labels to apply.
type LabelConfig struct {
	ClearExisting bool    `yaml:"clear_existing"`
	Items         []Label `yaml:"items"`
}

// Label is a single GitHub label.
type Label struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
}

// BoilerplateConfig defines files to scaffold.
type BoilerplateConfig struct {
	License  string           `yaml:"license"`
	Gitignore string          `yaml:"gitignore"`
	Files    []BoilerplateFile `yaml:"files"`
}

// BoilerplateFile maps a template source to a destination path.
type BoilerplateFile struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

// BranchProtection defines branch protection rules.
type BranchProtection struct {
	Branch              string `yaml:"branch"`
	RequiredReviews     int    `yaml:"required_reviews"`
	DismissStaleReviews bool   `yaml:"dismiss_stale_reviews"`
	RequireStatusChecks bool   `yaml:"require_status_checks"`
}

// LoadFromFile loads config from a YAML file. If the file doesn't exist,
// returns the built-in default config.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if len(data) > maxConfigSize {
		return nil, fmt.Errorf("config file exceeds maximum size of %d bytes", maxConfigSize)
	}

	// Check file permissions
	info, err := os.Stat(path)
	if err == nil {
		perm := info.Mode().Perm()
		if perm&0077 != 0 {
			fmt.Fprintf(os.Stderr, "warning: config file %s has permissions %o, recommend 0600\n", path, perm)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// defaultConfig returns built-in defaults used when no config file exists.
func defaultConfig() *Config {
	return &Config{
		DefaultProfile: "personal",
		Profiles: map[string]Profile{
			"personal": {
				Description: "Personal project - minimal config",
				Settings: RepoSettings{
					HasWiki:             false,
					HasProjects:         false,
					DeleteBranchOnMerge: true,
					AllowSquashMerge:    true,
					AllowMergeCommit:    true,
				},
				Labels: LabelConfig{
					ClearExisting: true,
					Items: []Label{
						{Name: "bug", Color: "d73a4a"},
						{Name: "enhancement", Color: "a2eeef"},
						{Name: "chore", Color: "fef2c0"},
					},
				},
				Boilerplate: BoilerplateConfig{
					License: "MIT",
				},
			},
			"oss": {
				Description: "Open source project defaults",
				Settings: RepoSettings{
					HasWiki:             false,
					HasProjects:         false,
					DeleteBranchOnMerge: true,
					AllowSquashMerge:    true,
				},
				Labels: LabelConfig{
					ClearExisting: true,
					Items: []Label{
						{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
						{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
						{Name: "documentation", Color: "0075ca", Description: "Improvements or additions to docs"},
						{Name: "good first issue", Color: "7057ff", Description: "Good for newcomers"},
						{Name: "help wanted", Color: "008672", Description: "Extra attention is needed"},
						{Name: "wontfix", Color: "ffffff", Description: "This will not be worked on"},
					},
				},
				Boilerplate: BoilerplateConfig{
					License:   "MIT",
					Gitignore: "Go",
				},
				BranchProtection: BranchProtection{
					Branch:              "main",
					DismissStaleReviews: true,
				},
			},
			"action": {
				Description: "GitHub Action defaults",
				Settings: RepoSettings{
					HasWiki:             false,
					HasProjects:         false,
					DeleteBranchOnMerge: true,
					AllowSquashMerge:    true,
				},
				Labels: LabelConfig{
					ClearExisting: true,
					Items: []Label{
						{Name: "bug", Color: "d73a4a"},
						{Name: "enhancement", Color: "a2eeef"},
						{Name: "breaking change", Color: "e11d48", Description: "Introduces a breaking change"},
					},
				},
				Boilerplate: BoilerplateConfig{
					License: "MIT",
				},
			},
		},
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS — all 4 tests

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config types and YAML loader with built-in defaults"
```

---

### Task 3: Input Validation

**Files:**
- Create: `internal/config/validation.go`
- Create: `internal/config/validation_test.go`

**Step 1: Write the validation tests**

```go
package config

import "testing"

func TestValidateRepoName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-repo", false},
		{"valid dots", "my.repo", false},
		{"valid underscores", "my_repo", false},
		{"empty", "", true},
		{"spaces", "my repo", true},
		{"special chars", "my@repo", true},
		{"path traversal", "../evil", true},
		{"too long", string(make([]byte, 101)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepoName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelColor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase", "d73a4a", false},
		{"valid uppercase", "D73A4A", false},
		{"valid mixed", "aaBB11", false},
		{"too short", "d73a4", true},
		{"too long", "d73a4aa", true},
		{"invalid hex", "zzzzzz", true},
		{"with hash", "#d73a4a", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabelColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabelColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "oss", false},
		{"valid with dash", "my-profile", false},
		{"valid with underscore", "my_profile", false},
		{"empty", "", true},
		{"spaces", "my profile", true},
		{"special", "my@profile", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfileName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLabelName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "bug", false},
		{"valid with spaces", "good first issue", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 51)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabelName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLabelName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v -run TestValidate`
Expected: FAIL — functions not defined

**Step 3: Write the validation functions**

```go
package config

import (
	"fmt"
	"regexp"
	"unicode"
)

var (
	repoNamePattern    = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	labelColorPattern  = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
	profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func ValidateRepoName(name string) error {
	if name == "" {
		return fmt.Errorf("repo name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("repo name cannot exceed 100 characters")
	}
	if !repoNamePattern.MatchString(name) {
		return fmt.Errorf("repo name %q contains invalid characters (allowed: a-z, 0-9, '.', '_', '-')", name)
	}
	return nil
}

func ValidateLabelColor(color string) error {
	if !labelColorPattern.MatchString(color) {
		return fmt.Errorf("label color %q must be a 6-character hex string (e.g., d73a4a)", color)
	}
	return nil
}

func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("profile name %q contains invalid characters (allowed: a-z, 0-9, '_', '-')", name)
	}
	return nil
}

func ValidateLabelName(name string) error {
	if name == "" {
		return fmt.Errorf("label name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("label name cannot exceed 50 characters")
	}
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return fmt.Errorf("label name contains non-printable characters")
		}
	}
	return nil
}

// ValidateProfile validates all fields of a profile.
func ValidateProfile(name string, p Profile) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	for _, l := range p.Labels.Items {
		if err := ValidateLabelName(l.Name); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		if err := ValidateLabelColor(l.Color); err != nil {
			return fmt.Errorf("profile %q label %q: %w", name, l.Name, err)
		}
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v -run TestValidate`
Expected: PASS — all tests

**Step 5: Commit**

```bash
git add internal/config/validation.go internal/config/validation_test.go
git commit -m "feat: add input validation for repo names, labels, profiles"
```

---

### Task 4: GitHub Client (gh CLI Wrapper)

**Files:**
- Create: `internal/github/client.go`
- Create: `internal/github/client_test.go`

**Step 1: Write the test**

```go
package github

import (
	"testing"
)

func TestBuildArgs_NoShellInterpolation(t *testing.T) {
	// Verify that arguments are separate strings, not shell-interpolated.
	c := NewClient()
	args := c.buildArgs("api", "repos/owner/repo", "-X", "PATCH")
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "api" {
		t.Errorf("args[0] = %q, want %q", args[0], "api")
	}
}

func TestCheckGHInstalled(t *testing.T) {
	c := NewClient()
	err := c.CheckInstalled()
	// This test only passes if gh is installed on the test machine.
	// In CI, we'd skip this. For now, just verify no panic.
	if err != nil {
		t.Skipf("gh not installed: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/github/ -v`
Expected: FAIL — types not defined

**Step 3: Write the client**

```go
package github

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps the gh CLI for GitHub API interactions.
// All commands use exec.Command with argument arrays — never shell interpolation.
type Client struct {
	ghPath string
}

// NewClient creates a new GitHub client that shells out to gh.
func NewClient() *Client {
	return &Client{ghPath: "gh"}
}

// CheckInstalled verifies that gh is installed and authenticated.
func (c *Client) CheckInstalled() error {
	if _, err := exec.LookPath(c.ghPath); err != nil {
		return fmt.Errorf("gh CLI not found. Install it from https://cli.github.com")
	}
	out, err := c.run("auth", "status")
	if err != nil {
		return fmt.Errorf("gh auth failed: %s", out)
	}
	return nil
}

// buildArgs constructs the argument list for exec.Command.
func (c *Client) buildArgs(args ...string) []string {
	return args
}

// run executes a gh command and returns combined output.
func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command(c.ghPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(stderr.String()), fmt.Errorf("gh %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunJSON executes a gh command and returns the raw JSON output.
func (c *Client) RunJSON(args ...string) (string, error) {
	return c.run(args...)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/github/ -v`
Expected: PASS (or skip if gh not installed)

**Step 5: Commit**

```bash
git add internal/github/client.go internal/github/client_test.go
git commit -m "feat: add gh CLI wrapper with command injection prevention"
```

---

### Task 5: Repo Operations (Create & Settings)

**Files:**
- Create: `internal/github/repo.go`
- Create: `internal/github/repo_test.go`

**Step 1: Write the test**

```go
package github

import (
	"strings"
	"testing"
)

func TestCreateRepoArgs(t *testing.T) {
	c := NewClient()
	args := c.createRepoArgs("my-tool", "A CLI tool", true)
	// Verify args are structured correctly for exec.Command
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "repo") || !strings.Contains(joined, "create") {
		t.Errorf("missing repo create in args: %v", args)
	}
	if !strings.Contains(joined, "--public") {
		t.Errorf("missing --public flag: %v", args)
	}
	if !strings.Contains(joined, "my-tool") {
		t.Errorf("missing repo name: %v", args)
	}
}

func TestCreateRepoArgs_Private(t *testing.T) {
	c := NewClient()
	args := c.createRepoArgs("my-tool", "A CLI tool", false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--private") {
		t.Errorf("missing --private flag: %v", args)
	}
}

func TestUpdateSettingsArgs(t *testing.T) {
	c := NewClient()
	args := c.updateSettingsArgs("owner/repo", map[string]interface{}{
		"has_wiki":     false,
		"has_projects": false,
	})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "api") {
		t.Errorf("missing api command: %v", args)
	}
	if !strings.Contains(joined, "PATCH") {
		t.Errorf("missing PATCH method: %v", args)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/github/ -v -run TestCreateRepo`
Expected: FAIL — methods not defined

**Step 3: Write repo operations**

```go
package github

import (
	"encoding/json"
	"fmt"
)

// createRepoArgs builds args for `gh repo create`. Exported for testing.
func (c *Client) createRepoArgs(name, description string, public bool) []string {
	visibility := "--private"
	if public {
		visibility = "--public"
	}
	args := []string{"repo", "create", name, visibility, "--description", description}
	return args
}

// CreateRepo creates a new GitHub repository.
func (c *Client) CreateRepo(name, description string, public bool) (string, error) {
	args := c.createRepoArgs(name, description, public)
	out, err := c.run(args...)
	if err != nil {
		return "", fmt.Errorf("creating repo: %w", err)
	}
	return out, nil
}

// updateSettingsArgs builds args for `gh api` PATCH to update repo settings.
func (c *Client) updateSettingsArgs(nwo string, settings map[string]interface{}) []string {
	body, _ := json.Marshal(settings)
	return []string{"api", fmt.Sprintf("repos/%s", nwo), "-X", "PATCH", "--input", "-"}
}

// UpdateSettings updates repository settings via the GitHub API.
func (c *Client) UpdateSettings(nwo string, settings map[string]interface{}) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	args := []string{"api", fmt.Sprintf("repos/%s", nwo), "-X", "PATCH", "--input", "-"}
	cmd := c.commandWithStdin(args, body)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("updating settings: %w\n%s", err, string(out))
	}
	return nil
}

// commandWithStdin creates an exec.Cmd with stdin piped.
func (c *Client) commandWithStdin(args []string, stdin []byte) *execCmd {
	return newExecCmd(c.ghPath, args, stdin)
}
```

Note: The `commandWithStdin` helper needs a small abstraction. Simplify to use `exec.Command` directly:

```go
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func (c *Client) createRepoArgs(name, description string, public bool) []string {
	visibility := "--private"
	if public {
		visibility = "--public"
	}
	return []string{"repo", "create", name, visibility, "--description", description}
}

func (c *Client) CreateRepo(name, description string, public bool) (string, error) {
	args := c.createRepoArgs(name, description, public)
	out, err := c.run(args...)
	if err != nil {
		return "", fmt.Errorf("creating repo: %w", err)
	}
	return out, nil
}

func (c *Client) updateSettingsArgs(nwo string, settings map[string]interface{}) []string {
	body, _ := json.Marshal(settings)
	_ = body // used in UpdateSettings, not in args
	return []string{"api", fmt.Sprintf("repos/%s", nwo), "-X", "PATCH", "--input", "-"}
}

func (c *Client) UpdateSettings(nwo string, settings map[string]interface{}) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	cmd := exec.Command(c.ghPath, "api", fmt.Sprintf("repos/%s", nwo), "-X", "PATCH", "--input", "-")
	cmd.Stdin = bytes.NewReader(body)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("updating settings: %w\n%s", err, string(out))
	}
	return nil
}

// SettingsFromRepoSettings converts our config type to a map for the API.
func SettingsFromRepoSettings(s interface{}) map[string]interface{} {
	data, _ := json.Marshal(s)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/github/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/github/repo.go internal/github/repo_test.go
git commit -m "feat: add repo create and settings update operations"
```

---

### Task 6: Label Operations

**Files:**
- Create: `internal/github/labels.go`
- Create: `internal/github/labels_test.go`

**Step 1: Write the test**

```go
package github

import (
	"strings"
	"testing"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

func TestCreateLabelArgs(t *testing.T) {
	c := NewClient()
	label := config.Label{Name: "bug", Color: "d73a4a", Description: "Something isn't working"}
	args := c.createLabelArgs("owner/repo", label)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "label") || !strings.Contains(joined, "create") {
		t.Errorf("missing label create: %v", args)
	}
	if !strings.Contains(joined, "bug") {
		t.Errorf("missing label name: %v", args)
	}
	if !strings.Contains(joined, "d73a4a") {
		t.Errorf("missing color: %v", args)
	}
}

func TestDeleteLabelArgs(t *testing.T) {
	c := NewClient()
	args := c.deleteLabelArgs("owner/repo", "bug")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "label") || !strings.Contains(joined, "delete") {
		t.Errorf("missing label delete: %v", args)
	}
	if !strings.Contains(joined, "bug") {
		t.Errorf("missing label name: %v", args)
	}
	if !strings.Contains(joined, "--yes") {
		t.Errorf("missing --yes flag: %v", args)
	}
}

func TestListLabelsArgs(t *testing.T) {
	c := NewClient()
	args := c.listLabelsArgs("owner/repo")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "label") || !strings.Contains(joined, "list") {
		t.Errorf("missing label list: %v", args)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/github/ -v -run TestLabel`
Expected: FAIL

**Step 3: Write label operations**

```go
package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

func (c *Client) createLabelArgs(nwo string, label config.Label) []string {
	args := []string{"label", "create", label.Name, "--color", label.Color, "--repo", nwo}
	if label.Description != "" {
		args = append(args, "--description", label.Description)
	}
	return args
}

func (c *Client) deleteLabelArgs(nwo string, name string) []string {
	return []string{"label", "delete", name, "--repo", nwo, "--yes"}
}

func (c *Client) listLabelsArgs(nwo string) []string {
	return []string{"label", "list", "--repo", nwo, "--json", "name"}
}

// CreateLabel creates a single label on the repo.
func (c *Client) CreateLabel(nwo string, label config.Label) error {
	args := c.createLabelArgs(nwo, label)
	if _, err := c.run(args...); err != nil {
		return fmt.Errorf("creating label %q: %w", label.Name, err)
	}
	return nil
}

// DeleteLabel deletes a single label from the repo.
func (c *Client) DeleteLabel(nwo string, name string) error {
	args := c.deleteLabelArgs(nwo, name)
	if _, err := c.run(args...); err != nil {
		return fmt.Errorf("deleting label %q: %w", name, err)
	}
	return nil
}

// ListLabels returns the names of all labels on the repo.
func (c *Client) ListLabels(nwo string) ([]string, error) {
	args := c.listLabelsArgs(nwo)
	out, err := c.run(args...)
	if err != nil {
		return nil, fmt.Errorf("listing labels: %w", err)
	}
	var labels []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &labels); err != nil {
		return nil, fmt.Errorf("parsing labels: %w", err)
	}
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.Name
	}
	return names, nil
}

// SyncLabels clears existing labels (if configured) and creates new ones.
// Returns counts of deleted and created labels, plus any errors.
func (c *Client) SyncLabels(nwo string, cfg config.LabelConfig) (deleted int, created int, errs []error) {
	if cfg.ClearExisting {
		existing, err := c.ListLabels(nwo)
		if err != nil {
			errs = append(errs, err)
			return
		}
		for _, name := range existing {
			if err := c.DeleteLabel(nwo, name); err != nil {
				errs = append(errs, err)
			} else {
				deleted++
			}
		}
	}

	for _, label := range cfg.Items {
		if err := c.CreateLabel(nwo, label); err != nil {
			errs = append(errs, err)
		} else {
			created++
		}
	}
	return
}

// LabelSummary returns a human-readable summary of the label config.
func LabelSummary(cfg config.LabelConfig) string {
	names := make([]string, len(cfg.Items))
	for i, l := range cfg.Items {
		names[i] = l.Name
	}
	return fmt.Sprintf("%d labels (%s)", len(cfg.Items), strings.Join(names, ", "))
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/github/ -v -run TestLabel`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/github/labels.go internal/github/labels_test.go
git commit -m "feat: add label CRUD operations with sync support"
```

---

### Task 7: Branch Protection

**Files:**
- Create: `internal/github/protection.go`
- Create: `internal/github/protection_test.go`

**Step 1: Write the test**

```go
package github

import (
	"testing"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

func TestBranchProtectionPayload(t *testing.T) {
	bp := config.BranchProtection{
		Branch:              "main",
		RequiredReviews:     1,
		DismissStaleReviews: true,
		RequireStatusChecks: false,
	}
	payload := buildProtectionPayload(bp)
	reviews, ok := payload["required_pull_request_reviews"].(map[string]interface{})
	if !ok {
		t.Fatal("missing required_pull_request_reviews")
	}
	if reviews["dismiss_stale_reviews"] != true {
		t.Error("dismiss_stale_reviews should be true")
	}
	if reviews["required_approving_review_count"] != 1 {
		t.Errorf("required_approving_review_count = %v, want 1", reviews["required_approving_review_count"])
	}
}

func TestBranchProtectionPayload_NoReviews(t *testing.T) {
	bp := config.BranchProtection{
		Branch:          "main",
		RequiredReviews: 0,
	}
	payload := buildProtectionPayload(bp)
	if payload["required_pull_request_reviews"] != nil {
		t.Error("expected nil required_pull_request_reviews when RequiredReviews is 0")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/github/ -v -run TestBranchProtection`
Expected: FAIL

**Step 3: Write branch protection**

```go
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

func buildProtectionPayload(bp config.BranchProtection) map[string]interface{} {
	payload := map[string]interface{}{
		"enforce_admins":                false,
		"required_status_checks":        nil,
		"restrictions":                  nil,
		"required_pull_request_reviews": nil,
	}

	if bp.RequiredReviews > 0 {
		payload["required_pull_request_reviews"] = map[string]interface{}{
			"dismiss_stale_reviews":          bp.DismissStaleReviews,
			"required_approving_review_count": bp.RequiredReviews,
		}
	}

	if bp.RequireStatusChecks {
		payload["required_status_checks"] = map[string]interface{}{
			"strict":   true,
			"contexts": []string{},
		}
	}

	return payload
}

// SetBranchProtection applies branch protection rules.
func (c *Client) SetBranchProtection(nwo string, bp config.BranchProtection) error {
	if bp.Branch == "" {
		return nil // no branch protection configured
	}

	payload := buildProtectionPayload(bp)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling protection: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/branches/%s/protection", nwo, bp.Branch)
	cmd := exec.Command(c.ghPath, "api", endpoint, "-X", "PUT", "--input", "-")
	cmd.Stdin = bytes.NewReader(body)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("setting branch protection: %w\n%s", err, string(out))
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/github/ -v -run TestBranchProtection`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/github/protection.go internal/github/protection_test.go
git commit -m "feat: add branch protection API support"
```

---

### Task 8: Embedded Templates & Boilerplate Scaffolding

**Files:**
- Create: `templates/contributing.md`
- Create: `templates/ci.yml`
- Create: `templates/action.yml`
- Create: `templates/action-ci.yml`
- Create: `templates/action-release.yml`
- Create: `internal/scaffold/templates.go`
- Create: `internal/scaffold/templates_test.go`

**Step 1: Create the template files**

`templates/contributing.md`:
```markdown
# Contributing

Thank you for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -am 'Add my feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

## Code of Conduct

Please be respectful and constructive in all interactions.
```

`templates/ci.yml`:
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: echo "Add build steps here"
      - name: Test
        run: echo "Add test steps here"
```

`templates/action.yml`:
```yaml
name: 'My Action'
description: 'A GitHub Action'
inputs:
  example:
    description: 'An example input'
    required: false
    default: ''
runs:
  using: 'node20'
  main: 'dist/index.js'
```

`templates/action-ci.yml`:
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Test
        run: echo "Add test steps here"
```

`templates/action-release.yml`:
```yaml
name: Release
on:
  release:
    types: [published]
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: echo "Add build steps here"
```

**Step 2: Write the template resolver test**

```go
package scaffold

import (
	"testing"
)

func TestResolveTemplate_Embedded(t *testing.T) {
	content, err := ResolveTemplate("contributing.md", "")
	if err != nil {
		t.Fatalf("ResolveTemplate: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty content")
	}
}

func TestResolveTemplate_MissingEmbedded(t *testing.T) {
	_, err := ResolveTemplate("nonexistent.txt", "")
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestResolveTemplate_UserOverride(t *testing.T) {
	dir := t.TempDir()
	// Write a user override
	userFile := dir + "/contributing.md"
	if err := writeTestFile(userFile, "custom content"); err != nil {
		t.Fatal(err)
	}
	content, err := ResolveTemplate("contributing.md", dir)
	if err != nil {
		t.Fatalf("ResolveTemplate: %v", err)
	}
	if string(content) != "custom content" {
		t.Errorf("expected user override, got: %s", content)
	}
}

func TestResolveTemplate_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolveTemplate("../../etc/passwd", dir)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./internal/scaffold/ -v`
Expected: FAIL

**Step 4: Write the template resolver**

```go
package scaffold

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:../../templates
var embeddedTemplates embed.FS

// Note: the embed path above assumes this file is at internal/scaffold/templates.go
// and templates/ is at the repo root. Adjust if needed.
// Actually, Go embed paths are relative to the source file's directory.
// We need to embed from the repo root, so we'll use a different approach.

// We'll define the embed in a file at the repo root and import it.
// For simplicity, let's embed from the package level.
```

Actually, Go's `embed` directive requires paths relative to the source file. Since `internal/scaffold/templates.go` can't reach `../../templates` with embed, we need to define the embed at the root level.

**Revised approach:** Create `templates.go` at the repo root that embeds the templates, and import it from scaffold.

Create `templates/embed.go`:
```go
package templates

import "embed"

//go:embed *
var FS embed.FS
```

Then `internal/scaffold/templates.go`:
```go
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvns/gh-repo-defaults/templates"
)

// ResolveTemplate returns the content of a template file.
// If userDir is non-empty and contains the file, it takes precedence.
// Otherwise falls back to embedded templates.
func ResolveTemplate(name string, userDir string) ([]byte, error) {
	// Block path traversal
	if strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid template path: %q", name)
	}

	// Try user override first
	if userDir != "" {
		userPath := filepath.Join(userDir, name)
		absUser, err := filepath.Abs(userPath)
		if err == nil {
			absDir, _ := filepath.Abs(userDir)
			if strings.HasPrefix(absUser, absDir+string(filepath.Separator)) {
				if data, err := os.ReadFile(absUser); err == nil {
					return data, nil
				}
			}
		}
	}

	// Fall back to embedded
	data, err := templates.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("template %q not found: %w", name, err)
	}
	return data, nil
}

// Helper for tests
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/scaffold/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add templates/ internal/scaffold/
git commit -m "feat: add embedded templates with user override and path traversal prevention"
```

---

### Task 9: Boilerplate Scaffolding (Clone, Copy, Push)

**Files:**
- Create: `internal/scaffold/boilerplate.go`
- Create: `internal/scaffold/boilerplate_test.go`

**Step 1: Write the test**

```go
package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

func TestPrepareBoilerplate(t *testing.T) {
	dir := t.TempDir()
	cfg := config.BoilerplateConfig{
		Files: []config.BoilerplateFile{
			{Src: "contributing.md", Dest: "CONTRIBUTING.md"},
			{Src: "ci.yml", Dest: ".github/workflows/ci.yml"},
		},
	}
	files, err := PrepareBoilerplate(cfg, dir, "")
	if err != nil {
		t.Fatalf("PrepareBoilerplate: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Check files were written
	contribPath := filepath.Join(dir, "CONTRIBUTING.md")
	if _, err := os.Stat(contribPath); os.IsNotExist(err) {
		t.Error("CONTRIBUTING.md not created")
	}

	ciPath := filepath.Join(dir, ".github", "workflows", "ci.yml")
	if _, err := os.Stat(ciPath); os.IsNotExist(err) {
		t.Error(".github/workflows/ci.yml not created")
	}
}

func TestPrepareBoilerplate_PathTraversalInDest(t *testing.T) {
	dir := t.TempDir()
	cfg := config.BoilerplateConfig{
		Files: []config.BoilerplateFile{
			{Src: "contributing.md", Dest: "../../etc/evil"},
		},
	}
	_, err := PrepareBoilerplate(cfg, dir, "")
	if err == nil {
		t.Error("expected error for path traversal in dest")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/scaffold/ -v -run TestPrepare`
Expected: FAIL

**Step 3: Write boilerplate scaffolding**

```go
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

// PrepareBoilerplate resolves templates and writes them into targetDir.
// Returns the list of destination paths that were written.
func PrepareBoilerplate(cfg config.BoilerplateConfig, targetDir string, userTemplateDir string) ([]string, error) {
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolving target dir: %w", err)
	}

	var written []string

	for _, f := range cfg.Files {
		// Validate dest path
		if strings.Contains(f.Dest, "..") {
			return nil, fmt.Errorf("invalid destination path: %q", f.Dest)
		}
		destPath := filepath.Join(absTarget, f.Dest)
		absDest, err := filepath.Abs(destPath)
		if err != nil {
			return nil, fmt.Errorf("resolving dest path: %w", err)
		}
		if !strings.HasPrefix(absDest, absTarget+string(filepath.Separator)) && absDest != absTarget {
			return nil, fmt.Errorf("destination %q escapes target directory", f.Dest)
		}

		// Resolve template content
		content, err := ResolveTemplate(f.Src, userTemplateDir)
		if err != nil {
			return nil, fmt.Errorf("resolving template for %q: %w", f.Dest, err)
		}

		// Create parent directories
		destDir := filepath.Dir(absDest)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %q: %w", destDir, err)
		}

		// Write file
		if err := os.WriteFile(absDest, content, 0644); err != nil {
			return nil, fmt.Errorf("writing %q: %w", f.Dest, err)
		}
		written = append(written, f.Dest)
	}

	return written, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/scaffold/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/scaffold/boilerplate.go internal/scaffold/boilerplate_test.go
git commit -m "feat: add boilerplate scaffolding with path traversal prevention"
```

---

### Task 10: Profiles CLI Subcommand

**Files:**
- Create: `cmd/profiles.go`

**Step 1: Write profiles subcommand**

```go
package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gvns/gh-repo-defaults/internal/config"
	"github.com/gvns/gh-repo-defaults/internal/github"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage profiles",
}

var profilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tLABELS\tDEFAULT")
		for name, p := range cfg.Profiles {
			def := ""
			if name == cfg.DefaultProfile {
				def = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", name, p.Description, len(p.Labels.Items), def)
		}
		return w.Flush()
	},
}

var profilesShowCmd = &cobra.Command{
	Use:   "show [profile]",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		name := args[0]
		p, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		fmt.Printf("Profile: %s\n", name)
		fmt.Printf("Description: %s\n\n", p.Description)

		fmt.Println("Settings:")
		fmt.Printf("  Wiki: %v\n", p.Settings.HasWiki)
		fmt.Printf("  Projects: %v\n", p.Settings.HasProjects)
		fmt.Printf("  Delete branch on merge: %v\n", p.Settings.DeleteBranchOnMerge)
		fmt.Printf("  Allow squash merge: %v\n", p.Settings.AllowSquashMerge)
		fmt.Printf("  Allow merge commit: %v\n", p.Settings.AllowMergeCommit)
		fmt.Printf("  Allow rebase merge: %v\n", p.Settings.AllowRebaseMerge)
		fmt.Println()

		fmt.Printf("Labels (%d):\n", len(p.Labels.Items))
		for _, l := range p.Labels.Items {
			desc := ""
			if l.Description != "" {
				desc = " - " + l.Description
			}
			fmt.Printf("  #%s %s%s\n", l.Color, l.Name, desc)
		}
		fmt.Println()

		if p.Boilerplate.License != "" {
			fmt.Printf("License: %s\n", p.Boilerplate.License)
		}
		if p.Boilerplate.Gitignore != "" {
			fmt.Printf("Gitignore: %s\n", p.Boilerplate.Gitignore)
		}
		if len(p.Boilerplate.Files) > 0 {
			fmt.Println("Boilerplate files:")
			for _, f := range p.Boilerplate.Files {
				fmt.Printf("  %s -> %s\n", f.Src, f.Dest)
			}
		}

		if p.BranchProtection.Branch != "" {
			fmt.Printf("\nBranch protection: %s\n", p.BranchProtection.Branch)
			fmt.Printf("  Required reviews: %d\n", p.BranchProtection.RequiredReviews)
		}

		_ = github.LabelSummary // ensure import is used

		return nil
	},
}

func loadConfig() (*config.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := home + "/.config/gh-repo-defaults/config.yaml"
	return config.LoadFromFile(configPath)
}

func init() {
	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesShowCmd)
	rootCmd.AddCommand(profilesCmd)
}
```

**Step 2: Build and test manually**

Run:
```bash
go build -o gh-repo-defaults . && ./gh-repo-defaults profiles list
```
Expected: table with personal, oss, action profiles

Run:
```bash
./gh-repo-defaults profiles show oss
```
Expected: full profile details printed

**Step 3: Commit**

```bash
git add cmd/profiles.go
git commit -m "feat: add profiles list/show CLI subcommands"
```

---

### Task 11: Create Subcommand (Non-Interactive)

**Files:**
- Create: `cmd/create.go`
- Create: `internal/github/orchestrator.go`
- Create: `internal/github/orchestrator_test.go`

**Step 1: Write the orchestrator (coordinates all steps)**

The orchestrator ties together: create repo, apply settings, sync labels, scaffold boilerplate, set branch protection. It reports progress via a callback.

```go
package github

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gvns/gh-repo-defaults/internal/config"
	"github.com/gvns/gh-repo-defaults/internal/scaffold"
)

// StepStatus represents the outcome of a single step.
type StepStatus struct {
	Name    string
	Success bool
	Message string
	Err     error
}

// ProgressFunc is called after each step completes.
type ProgressFunc func(StepStatus)

// CreateOpts holds all options for creating a repo with defaults.
type CreateOpts struct {
	Name        string
	Description string
	Public      bool
	Profile     config.Profile
	Owner       string // optional, for org repos
	OnProgress  ProgressFunc
}

func (o *CreateOpts) nwo() string {
	if o.Owner != "" {
		return o.Owner + "/" + o.Name
	}
	return o.Name
}

func (o *CreateOpts) report(name string, err error) {
	if o.OnProgress == nil {
		return
	}
	s := StepStatus{Name: name, Success: err == nil}
	if err != nil {
		s.Err = err
		s.Message = err.Error()
	}
	o.OnProgress(s)
}

// CreateWithDefaults creates a repo and applies all profile defaults.
// Returns the repo URL on success.
func (c *Client) CreateWithDefaults(opts CreateOpts) (string, error) {
	// Step 1: Create repo
	repoArg := opts.Name
	if opts.Owner != "" {
		repoArg = opts.Owner + "/" + opts.Name
	}
	url, err := c.CreateRepo(repoArg, opts.Description, opts.Public)
	opts.report("Created repository", err)
	if err != nil {
		return "", err
	}

	nwo := opts.nwo()
	// If CreateRepo returned a URL, extract nwo from it
	// gh repo create returns the URL like https://github.com/owner/repo
	// We need owner/repo for API calls
	if url != "" && nwo == opts.Name {
		// Try to extract owner from the URL
		// Format: https://github.com/OWNER/REPO
		parts := splitRepoURL(url)
		if parts != "" {
			nwo = parts
		}
	}

	// Step 2: Apply settings
	settings := SettingsFromRepoSettings(opts.Profile.Settings)
	err = c.UpdateSettings(nwo, settings)
	opts.report("Applied repo settings", err)

	// Step 3: Sync labels
	deleted, created, labelErrs := c.SyncLabels(nwo, opts.Profile.Labels)
	var labelErr error
	if len(labelErrs) > 0 {
		labelErr = fmt.Errorf("%d label errors", len(labelErrs))
	}
	opts.report(fmt.Sprintf("Synced labels (-%d/+%d)", deleted, created), labelErr)

	// Step 4: Scaffold boilerplate
	if len(opts.Profile.Boilerplate.Files) > 0 {
		err = c.scaffoldAndPush(nwo, opts.Profile.Boilerplate, opts.Name)
		opts.report("Pushed boilerplate files", err)
	}

	// Step 5: Branch protection
	if opts.Profile.BranchProtection.Branch != "" {
		err = c.SetBranchProtection(nwo, opts.Profile.BranchProtection)
		opts.report("Set branch protection", err)
	}

	return url, nil
}

func (c *Client) scaffoldAndPush(nwo string, bp config.BoilerplateConfig, repoName string) error {
	tmpDir, err := os.MkdirTemp("", "gh-repo-defaults-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone the repo
	cloneDir := filepath.Join(tmpDir, repoName)
	if _, err := c.run("repo", "clone", nwo, cloneDir); err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}

	// Get user template dir
	home, _ := os.UserHomeDir()
	userTemplateDir := filepath.Join(home, ".config", "gh-repo-defaults", "templates")

	// Write boilerplate files
	if _, err := scaffold.PrepareBoilerplate(bp, cloneDir, userTemplateDir); err != nil {
		return fmt.Errorf("preparing boilerplate: %w", err)
	}

	// Git add, commit, push
	gitCmd := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = cloneDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %v: %w\n%s", args, err, string(out))
		}
		return nil
	}
	if err := gitCmd("add", "-A"); err != nil {
		return err
	}
	if err := gitCmd("commit", "-m", "chore: add boilerplate files"); err != nil {
		return err
	}
	if err := gitCmd("push"); err != nil {
		return err
	}
	return nil
}

func splitRepoURL(url string) string {
	// Extract "owner/repo" from "https://github.com/owner/repo"
	// Simple string parsing
	prefix := "https://github.com/"
	if len(url) > len(prefix) && url[:len(prefix)] == prefix {
		return url[len(prefix):]
	}
	return ""
}

// SettingsFromRepoSettings converts config settings to a map for the API.
func SettingsFromRepoSettings(s config.RepoSettings) map[string]interface{} {
	data, _ := json.Marshal(s)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}
```

Note: Move `SettingsFromRepoSettings` from `repo.go` to `orchestrator.go` (or keep in `repo.go` and import — the point is it should only be defined once). Adjust `repo.go` accordingly to avoid duplication.

**Step 2: Write the create subcommand**

```go
package cmd

import (
	"fmt"

	"github.com/gvns/gh-repo-defaults/internal/config"
	ghclient "github.com/gvns/gh-repo-defaults/internal/github"
	"github.com/spf13/cobra"
)

var (
	createProfile string
	createPublic  bool
	createPrivate bool
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new repo with profile defaults",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := config.ValidateRepoName(name); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		profileName := createProfile
		if profileName == "" {
			profileName = cfg.DefaultProfile
		}
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}
		if err := config.ValidateProfile(profileName, profile); err != nil {
			return err
		}

		public := createPublic
		if createPrivate {
			public = false
		}

		client := ghclient.NewClient()
		if err := client.CheckInstalled(); err != nil {
			return err
		}

		opts := ghclient.CreateOpts{
			Name:        name,
			Description: cmd.Flag("description").Value.String(),
			Public:      public,
			Profile:     profile,
			Owner:       cfg.DefaultOwner,
			OnProgress: func(s ghclient.StepStatus) {
				if s.Success {
					fmt.Printf("  ✓ %s\n", s.Name)
				} else {
					fmt.Printf("  ✗ %s: %s\n", s.Name, s.Message)
				}
			},
		}

		url, err := client.CreateWithDefaults(opts)
		if err != nil {
			return err
		}
		fmt.Printf("\nDone! %s\n", url)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createProfile, "profile", "p", "", "Profile to apply (default: from config)")
	createCmd.Flags().BoolVar(&createPublic, "public", false, "Create public repo")
	createCmd.Flags().BoolVar(&createPrivate, "private", false, "Create private repo")
	createCmd.Flags().String("description", "", "Repo description")
	rootCmd.AddCommand(createCmd)
}
```

**Step 3: Build and verify**

Run: `go build -o gh-repo-defaults .`
Expected: compiles cleanly

Run: `./gh-repo-defaults create --help`
Expected: shows usage with --profile, --public, --private, --description flags

**Step 4: Commit**

```bash
git add cmd/create.go internal/github/orchestrator.go
git commit -m "feat: add create subcommand with orchestrated repo setup"
```

---

### Task 12: Apply Subcommand (Existing Repos)

**Files:**
- Create: `cmd/apply.go`

**Step 1: Write the apply subcommand**

```go
package cmd

import (
	"fmt"

	"github.com/gvns/gh-repo-defaults/internal/config"
	ghclient "github.com/gvns/gh-repo-defaults/internal/github"
	"github.com/spf13/cobra"
)

var applyProfile string

var applyCmd = &cobra.Command{
	Use:   "apply [owner/repo]",
	Short: "Apply a profile to an existing repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nwo := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		profileName := applyProfile
		if profileName == "" {
			profileName = cfg.DefaultProfile
		}
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}
		if err := config.ValidateProfile(profileName, profile); err != nil {
			return err
		}

		client := ghclient.NewClient()
		if err := client.CheckInstalled(); err != nil {
			return err
		}

		progress := func(s ghclient.StepStatus) {
			if s.Success {
				fmt.Printf("  ✓ %s\n", s.Name)
			} else {
				fmt.Printf("  ✗ %s: %s\n", s.Name, s.Message)
			}
		}

		fmt.Printf("Applying profile %q to %s...\n", profileName, nwo)

		// Apply settings
		settings := ghclient.SettingsFromRepoSettings(profile.Settings)
		err = client.UpdateSettings(nwo, settings)
		progress(ghclient.StepStatus{Name: "Applied repo settings", Success: err == nil, Err: err})

		// Sync labels
		deleted, created, labelErrs := client.SyncLabels(nwo, profile.Labels)
		var labelErr error
		if len(labelErrs) > 0 {
			labelErr = fmt.Errorf("%d label errors", len(labelErrs))
		}
		progress(ghclient.StepStatus{
			Name:    fmt.Sprintf("Synced labels (-%d/+%d)", deleted, created),
			Success: labelErr == nil,
			Err:     labelErr,
		})

		// Branch protection
		if profile.BranchProtection.Branch != "" {
			err = client.SetBranchProtection(nwo, profile.BranchProtection)
			progress(ghclient.StepStatus{Name: "Set branch protection", Success: err == nil, Err: err})
		}

		fmt.Println("\nDone!")
		return nil
	},
}

func init() {
	applyCmd.Flags().StringVarP(&applyProfile, "profile", "p", "", "Profile to apply (default: from config)")
	rootCmd.AddCommand(applyCmd)
}
```

**Step 2: Build and verify**

Run: `go build -o gh-repo-defaults . && ./gh-repo-defaults apply --help`
Expected: shows usage

**Step 3: Commit**

```bash
git add cmd/apply.go
git commit -m "feat: add apply subcommand for existing repos"
```

---

### Task 13: TUI — Styles & App Shell

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/app.go`

**Step 1: Write styles**

```go
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	indigo = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	red    = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	subtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
)

type Styles struct {
	Base,
	Header,
	Success,
	Error,
	Help,
	Highlight lipgloss.Style
}

func NewStyles(lg *lipgloss.Renderer) *Styles {
	s := Styles{}
	s.Base = lg.NewStyle().Padding(1, 2)
	s.Header = lg.NewStyle().Foreground(indigo).Bold(true).Padding(0, 1)
	s.Success = lg.NewStyle().Foreground(green)
	s.Error = lg.NewStyle().Foreground(red)
	s.Help = lg.NewStyle().Foreground(lipgloss.Color("240"))
	s.Highlight = lg.NewStyle().Foreground(lipgloss.Color("212"))
	return &s
}
```

**Step 2: Write the app shell (screen router)**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gvns/gh-repo-defaults/internal/config"
)

type screen int

const (
	screenMode screen = iota
	screenCreate
	screenProgress
)

type App struct {
	screen  screen
	styles  *Styles
	lg      *lipgloss.Renderer
	config  *config.Config
	width   int

	// Sub-models
	mode     ModeModel
	create   CreateModel
	progress ProgressModel
}

func NewApp(cfg *config.Config) App {
	lg := lipgloss.DefaultRenderer()
	styles := NewStyles(lg)
	a := App{
		screen: screenMode,
		styles: styles,
		lg:     lg,
		config: cfg,
		width:  80,
	}
	a.mode = NewModeModel(styles)
	return a
}

func (a App) Init() tea.Cmd {
	return a.mode.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Interrupt
		}
	}

	switch a.screen {
	case screenMode:
		return a.updateMode(msg)
	case screenCreate:
		return a.updateCreate(msg)
	case screenProgress:
		return a.updateProgress(msg)
	}
	return a, nil
}

func (a App) View() string {
	switch a.screen {
	case screenMode:
		return a.mode.View()
	case screenCreate:
		return a.create.View()
	case screenProgress:
		return a.progress.View()
	}
	return ""
}
```

Note: `updateMode`, `updateCreate`, `updateProgress`, and sub-models (`ModeModel`, `CreateModel`, `ProgressModel`) will be implemented in the next tasks. For now, create stubs.

**Step 3: Commit**

```bash
git add internal/tui/styles.go internal/tui/app.go
git commit -m "feat: add TUI styles and app shell with screen routing"
```

---

### Task 14: TUI — Mode Selection Screen

**Files:**
- Create: `internal/tui/mode.go`

**Step 1: Write mode selection using huh**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type modeChoice string

const (
	modeCreate   modeChoice = "create"
	modeApply    modeChoice = "apply"
	modeProfiles modeChoice = "profiles"
)

type ModeModel struct {
	form   *huh.Form
	choice modeChoice
	styles *Styles
}

func NewModeModel(styles *Styles) ModeModel {
	var choice modeChoice
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[modeChoice]().
				Key("mode").
				Title("gh repo-defaults").
				Description("What would you like to do?").
				Options(
					huh.NewOption("Create new repo", modeCreate),
					huh.NewOption("Apply to existing repo", modeApply),
					huh.NewOption("Manage profiles", modeProfiles),
				).
				Value(&choice),
		),
	).WithShowHelp(false)

	return ModeModel{
		form:   f,
		styles: styles,
	}
}

func (m ModeModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m ModeModel) Update(msg tea.Msg) (ModeModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m ModeModel) View() string {
	return m.form.View()
}

func (m ModeModel) Done() bool {
	return m.form.State == huh.StateCompleted
}

func (m ModeModel) Choice() modeChoice {
	return m.choice
}
```

**Step 2: Wire into app.go — add updateMode**

Add to `app.go`:

```go
// modeSelectedMsg signals the mode screen completed.
type modeSelectedMsg struct{ choice modeChoice }

func (a App) updateMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	mode, cmd := a.mode.Update(msg)
	a.mode = mode

	if a.mode.Done() {
		choice := a.mode.Choice()
		switch choice {
		case modeCreate:
			a.screen = screenCreate
			a.create = NewCreateModel(a.styles, a.config)
			return a, a.create.Init()
		case modeProfiles:
			return a, tea.Quit // Fall back to CLI for profiles
		}
	}
	return a, cmd
}
```

**Step 3: Build and verify**

Run: `go build -o gh-repo-defaults .`
Expected: compiles

**Step 4: Commit**

```bash
git add internal/tui/mode.go internal/tui/app.go
git commit -m "feat: add TUI mode selection screen"
```

---

### Task 15: TUI — Create Repo Form Screen

**Files:**
- Create: `internal/tui/create.go`

**Step 1: Write the create form using huh**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/gvns/gh-repo-defaults/internal/config"
)

type CreateModel struct {
	form        *huh.Form
	styles      *Styles
	cfg         *config.Config
	Name        string
	Description string
	Visibility  string
	ProfileName string
}

func NewCreateModel(styles *Styles, cfg *config.Config) CreateModel {
	m := CreateModel{styles: styles, cfg: cfg}

	// Build profile options
	profileOpts := make([]huh.Option[string], 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		profileOpts = append(profileOpts, huh.NewOption(name, name))
	}

	m.ProfileName = cfg.DefaultProfile
	m.Visibility = "public"

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Repository name").
				Validate(func(s string) error {
					return config.ValidateRepoName(s)
				}).
				Value(&m.Name),

			huh.NewInput().
				Key("description").
				Title("Description").
				Value(&m.Description),

			huh.NewSelect[string]().
				Key("visibility").
				Title("Visibility").
				Options(
					huh.NewOption("Public", "public"),
					huh.NewOption("Private", "private"),
				).
				Value(&m.Visibility),

			huh.NewSelect[string]().
				Key("profile").
				Title("Profile").
				Options(profileOpts...).
				Value(&m.ProfileName),

			huh.NewConfirm().
				Key("confirm").
				Title("Create repository?").
				Affirmative("Create").
				Negative("Back"),
		),
	).WithShowHelp(true)

	return m
}

func (m CreateModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m CreateModel) Update(msg tea.Msg) (CreateModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m CreateModel) View() string {
	return m.form.View()
}

func (m CreateModel) Done() bool {
	return m.form.State == huh.StateCompleted
}

func (m CreateModel) Confirmed() bool {
	return m.form.GetBool("confirm")
}

func (m CreateModel) IsPublic() bool {
	return m.Visibility == "public"
}

func (m CreateModel) Profile() config.Profile {
	return m.cfg.Profiles[m.ProfileName]
}
```

**Step 2: Wire into app.go — add updateCreate**

```go
func (a App) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	create, cmd := a.create.Update(msg)
	a.create = create

	if a.create.Done() {
		if !a.create.Confirmed() {
			// Go back to mode selection
			a.screen = screenMode
			a.mode = NewModeModel(a.styles)
			return a, a.mode.Init()
		}
		// Start creating
		a.screen = screenProgress
		a.progress = NewProgressModel(a.styles, a.create)
		return a, a.progress.Init()
	}
	return a, cmd
}
```

**Step 3: Build and verify**

Run: `go build -o gh-repo-defaults .`
Expected: compiles

**Step 4: Commit**

```bash
git add internal/tui/create.go internal/tui/app.go
git commit -m "feat: add TUI create repo form screen"
```

---

### Task 16: TUI — Progress Screen

**Files:**
- Create: `internal/tui/progress.go`

**Step 1: Write the progress screen**

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	ghclient "github.com/gvns/gh-repo-defaults/internal/github"
)

type stepResult struct {
	status ghclient.StepStatus
}

type allDoneMsg struct {
	url string
	err error
}

type ProgressModel struct {
	styles  *Styles
	create  CreateModel
	steps   []ghclient.StepStatus
	done    bool
	url     string
	err     error
}

func NewProgressModel(styles *Styles, create CreateModel) ProgressModel {
	return ProgressModel{
		styles: styles,
		create: create,
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return m.startCreation()
}

func (m ProgressModel) startCreation() tea.Cmd {
	return func() tea.Msg {
		client := ghclient.NewClient()
		var steps []ghclient.StepStatus

		opts := ghclient.CreateOpts{
			Name:        m.create.Name,
			Description: m.create.Description,
			Public:      m.create.IsPublic(),
			Profile:     m.create.Profile(),
			OnProgress: func(s ghclient.StepStatus) {
				steps = append(steps, s)
			},
		}

		url, err := client.CreateWithDefaults(opts)
		return allDoneMsg{url: url, err: err}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case allDoneMsg:
		m.done = true
		m.url = msg.url
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m ProgressModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Creating repository..."))
	b.WriteString("\n\n")

	if !m.done {
		b.WriteString("  Working...\n")
	} else {
		if m.err != nil {
			b.WriteString(m.styles.Error.Render(fmt.Sprintf("  Error: %s\n", m.err)))
		}
		if m.url != "" {
			b.WriteString(m.styles.Success.Render(fmt.Sprintf("\n  Done! %s\n", m.url)))
		}
		b.WriteString("\n  Press q to exit.\n")
	}

	return b.String()
}

func (m ProgressModel) Done() bool {
	return m.done
}
```

**Step 2: Wire into app.go — add updateProgress**

```go
func (a App) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.progress.Done() && (msg.String() == "q" || msg.String() == "enter") {
			return a, tea.Quit
		}
	}
	progress, cmd := a.progress.Update(msg)
	a.progress = progress
	return a, cmd
}
```

**Step 3: Wire TUI into root command**

Update `cmd/root.go` to launch TUI when no subcommand given:

```go
package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gvns/gh-repo-defaults/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-repo-defaults",
	Short: "Create GitHub repos with consistent defaults",
	Long:  "A gh CLI extension that creates GitHub repos and applies labels, settings, boilerplate, and branch protection from named profiles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		app := tui.NewApp(cfg)
		p := tea.NewProgram(app)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
```

**Step 4: Build and verify**

Run: `go build -o gh-repo-defaults .`
Expected: compiles. Running `./gh-repo-defaults` launches the TUI.

**Step 5: Commit**

```bash
git add internal/tui/progress.go internal/tui/app.go cmd/root.go
git commit -m "feat: add TUI progress screen and wire up full TUI flow"
```

---

### Task 17: gh Extension Metadata & Installation

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Ensure the binary name matches gh extension convention**

The binary must be named `gh-repo-defaults` (which it already is). Verify:

Run: `go build -o gh-repo-defaults . && ./gh-repo-defaults --help`
Expected: help output

**Step 2: Test local extension installation**

Run: `gh extension install .`
Expected: extension installed locally

Run: `gh repo-defaults --help`
Expected: help output via gh

**Step 3: Create release workflow**

`.github/workflows/release.yml`:
```yaml
name: Release
on:
  push:
    tags:
      - 'v*'
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: cli/gh-extension-precompile@v2
```

**Step 4: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "feat: add release workflow for gh extension distribution"
```

---

### Task 18: Integration Test & Polish

**Files:**
- Create: `internal/github/orchestrator_test.go`

**Step 1: Write integration-style test for orchestrator arg building**

```go
package github

import (
	"testing"

	"github.com/gvns/gh-repo-defaults/internal/config"
)

func TestSettingsFromRepoSettings(t *testing.T) {
	s := config.RepoSettings{
		HasWiki:             false,
		HasProjects:         false,
		DeleteBranchOnMerge: true,
		AllowSquashMerge:    true,
	}
	m := SettingsFromRepoSettings(s)
	if m["has_wiki"] != false {
		t.Error("has_wiki should be false")
	}
	if m["delete_branch_on_merge"] != true {
		t.Error("delete_branch_on_merge should be true")
	}
}

func TestSplitRepoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/owner/repo", "owner/repo"},
		{"https://github.com/org/my-tool", "org/my-tool"},
		{"something-else", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := splitRepoURL(tt.input)
		if got != tt.want {
			t.Errorf("splitRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: all PASS

**Step 3: Run go vet and format check**

Run: `go vet ./... && gofmt -l .`
Expected: no issues

**Step 4: Commit**

```bash
git add internal/github/orchestrator_test.go
git commit -m "test: add orchestrator unit tests"
```

---

### Task 19: Final Build & Manual Smoke Test

**Step 1: Clean build**

Run: `go build -o gh-repo-defaults .`
Expected: binary built

**Step 2: Test CLI commands**

Run:
```bash
./gh-repo-defaults --help
./gh-repo-defaults profiles list
./gh-repo-defaults profiles show oss
./gh-repo-defaults create --help
./gh-repo-defaults apply --help
```
Expected: all output correct

**Step 3: Test TUI launches**

Run: `./gh-repo-defaults`
Expected: TUI launches with mode selection

**Step 4: Run full test suite one final time**

Run: `go test ./... -v -count=1`
Expected: all PASS

**Step 5: Final commit (if any fixes needed)**

```bash
git add -A && git commit -m "chore: final polish"
```
