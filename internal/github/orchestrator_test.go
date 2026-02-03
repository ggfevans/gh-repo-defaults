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
