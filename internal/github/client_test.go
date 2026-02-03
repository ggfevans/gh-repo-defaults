package github

import (
	"testing"
)

func TestBuildArgs_NoShellInterpolation(t *testing.T) {
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
	if err != nil {
		t.Skipf("gh not installed: %v", err)
	}
}
