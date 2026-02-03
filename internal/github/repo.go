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
