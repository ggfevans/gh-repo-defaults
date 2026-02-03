package github

import (
	"strings"
	"testing"
)

func TestCreateRepoArgs(t *testing.T) {
	c := NewClient()
	args := c.createRepoArgs("my-tool", "A CLI tool", true)
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
