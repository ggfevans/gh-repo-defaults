package cmd

import (
	"fmt"

	"github.com/gvns/gh-repo-defaults/internal/config"
	ghclient "github.com/gvns/gh-repo-defaults/internal/github"
	"github.com/spf13/cobra"
)

var (
	createProfile string
	createPublic  bool
	createPrivate bool
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new repo with profile defaults",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := config.ValidateRepoName(name); err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		profileName := createProfile
		if profileName == "" {
			profileName = cfg.DefaultProfile
		}
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}
		if err := config.ValidateProfile(profileName, profile); err != nil {
			return err
		}

		public := createPublic
		if createPrivate {
			public = false
		}

		client := ghclient.NewClient()
		if err := client.CheckInstalled(); err != nil {
			return err
		}

		opts := ghclient.CreateOpts{
			Name:        name,
			Description: cmd.Flag("description").Value.String(),
			Public:      public,
			Profile:     profile,
			Owner:       cfg.DefaultOwner,
			OnProgress: func(s ghclient.StepStatus) {
				if s.Success {
					fmt.Printf("  ✓ %s\n", s.Name)
				} else {
					fmt.Printf("  ✗ %s: %s\n", s.Name, s.Message)
				}
			},
		}

		url, err := client.CreateWithDefaults(opts)
		if err != nil {
			return err
		}
		fmt.Printf("\nDone! %s\n", url)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createProfile, "profile", "p", "", "Profile to apply (default: from config)")
	createCmd.Flags().BoolVar(&createPublic, "public", false, "Create public repo")
	createCmd.Flags().BoolVar(&createPrivate, "private", false, "Create private repo")
	createCmd.Flags().String("description", "", "Repo description")
	rootCmd.AddCommand(createCmd)
}
