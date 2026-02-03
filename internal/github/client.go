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

func NewClient() *Client {
	return &Client{ghPath: "gh"}
}

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

func (c *Client) buildArgs(args ...string) []string {
	return args
}

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

func (c *Client) RunJSON(args ...string) (string, error) {
	return c.run(args...)
}
