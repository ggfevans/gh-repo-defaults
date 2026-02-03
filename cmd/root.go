package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gvns/gh-repo-defaults/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-repo-defaults",
	Short: "Create GitHub repos with consistent defaults",
	Long:  "A gh CLI extension that creates GitHub repos and applies labels, settings, boilerplate, and branch protection from named profiles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		app := tui.NewApp(cfg)
		p := tea.NewProgram(app)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
