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
