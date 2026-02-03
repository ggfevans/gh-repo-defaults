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
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
}
