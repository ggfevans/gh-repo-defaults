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
