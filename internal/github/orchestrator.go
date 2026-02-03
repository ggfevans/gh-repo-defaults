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
func (c *Client) CreateWithDefaults(opts CreateOpts) (string, error) {
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
	if url != "" && nwo == opts.Name {
		parts := splitRepoURL(url)
		if parts != "" {
			nwo = parts
		}
	}

	// Apply settings
	settings := SettingsFromRepoSettings(opts.Profile.Settings)
	err = c.UpdateSettings(nwo, settings)
	opts.report("Applied repo settings", err)

	// Sync labels
	deleted, created, labelErrs := c.SyncLabels(nwo, opts.Profile.Labels)
	var labelErr error
	if len(labelErrs) > 0 {
		labelErr = fmt.Errorf("%d label errors", len(labelErrs))
	}
	opts.report(fmt.Sprintf("Synced labels (-%d/+%d)", deleted, created), labelErr)

	// Scaffold boilerplate
	if len(opts.Profile.Boilerplate.Files) > 0 {
		err = c.scaffoldAndPush(nwo, opts.Profile.Boilerplate, opts.Name)
		opts.report("Pushed boilerplate files", err)
	}

	// Branch protection
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

	cloneDir := filepath.Join(tmpDir, repoName)
	if _, err := c.run("repo", "clone", nwo, cloneDir); err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}

	home, _ := os.UserHomeDir()
	userTemplateDir := filepath.Join(home, ".config", "gh-repo-defaults", "templates")

	if _, err := scaffold.PrepareBoilerplate(bp, cloneDir, userTemplateDir); err != nil {
		return fmt.Errorf("preparing boilerplate: %w", err)
	}

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
