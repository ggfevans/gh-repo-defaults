package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

const maxConfigSize = 1024 * 1024 // 1MB

type Config struct {
	DefaultProfile string             `yaml:"default_profile"`
	DefaultOwner   string             `yaml:"default_owner"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

type Profile struct {
	Description      string            `yaml:"description"`
	Settings         RepoSettings      `yaml:"settings"`
	Labels           LabelConfig       `yaml:"labels"`
	Boilerplate      BoilerplateConfig `yaml:"boilerplate"`
	BranchProtection BranchProtection  `yaml:"branch_protection"`
}

type RepoSettings struct {
	HasWiki                  bool   `yaml:"has_wiki" json:"has_wiki"`
	HasProjects              bool   `yaml:"has_projects" json:"has_projects"`
	HasDiscussions           bool   `yaml:"has_discussions" json:"has_discussions"`
	DeleteBranchOnMerge      bool   `yaml:"delete_branch_on_merge" json:"delete_branch_on_merge"`
	AllowSquashMerge         bool   `yaml:"allow_squash_merge" json:"allow_squash_merge"`
	AllowMergeCommit         bool   `yaml:"allow_merge_commit" json:"allow_merge_commit"`
	AllowRebaseMerge         bool   `yaml:"allow_rebase_merge" json:"allow_rebase_merge"`
	SquashMergeCommitTitle   string `yaml:"squash_merge_commit_title" json:"squash_merge_commit_title,omitempty"`
	SquashMergeCommitMessage string `yaml:"squash_merge_commit_message" json:"squash_merge_commit_message,omitempty"`
}

type LabelConfig struct {
	ClearExisting bool    `yaml:"clear_existing"`
	Items         []Label `yaml:"items"`
}

type Label struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
}

type BoilerplateConfig struct {
	License   string            `yaml:"license"`
	Gitignore string            `yaml:"gitignore"`
	Files     []BoilerplateFile `yaml:"files"`
}

type BoilerplateFile struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

type BranchProtection struct {
	Branch              string `yaml:"branch"`
	RequiredReviews     int    `yaml:"required_reviews"`
	DismissStaleReviews bool   `yaml:"dismiss_stale_reviews"`
	RequireStatusChecks bool   `yaml:"require_status_checks"`
}

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

func defaultConfig() *Config {
	return &Config{
		DefaultProfile: "personal",
		Profiles: map[string]Profile{
			"personal": {
				Description: "Personal project - minimal config",
				Settings: RepoSettings{
					HasWiki: false, HasProjects: false,
					DeleteBranchOnMerge: true, AllowSquashMerge: true, AllowMergeCommit: true,
				},
				Labels: LabelConfig{
					ClearExisting: true,
					Items: []Label{
						{Name: "bug", Color: "d73a4a"},
						{Name: "enhancement", Color: "a2eeef"},
						{Name: "chore", Color: "fef2c0"},
					},
				},
				Boilerplate: BoilerplateConfig{License: "MIT"},
			},
			"oss": {
				Description: "Open source project defaults",
				Settings: RepoSettings{
					HasWiki: false, HasProjects: false,
					DeleteBranchOnMerge: true, AllowSquashMerge: true,
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
				Boilerplate:      BoilerplateConfig{License: "MIT", Gitignore: "Go"},
				BranchProtection: BranchProtection{Branch: "main", DismissStaleReviews: true},
			},
			"action": {
				Description: "GitHub Action defaults",
				Settings: RepoSettings{
					HasWiki: false, HasProjects: false,
					DeleteBranchOnMerge: true, AllowSquashMerge: true,
				},
				Labels: LabelConfig{
					ClearExisting: true,
					Items: []Label{
						{Name: "bug", Color: "d73a4a"},
						{Name: "enhancement", Color: "a2eeef"},
						{Name: "breaking change", Color: "e11d48", Description: "Introduces a breaking change"},
					},
				},
				Boilerplate: BoilerplateConfig{License: "MIT"},
			},
		},
	}
}
