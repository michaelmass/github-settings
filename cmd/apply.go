package cmd

import (
	"github.com/michaelmass/github-settings/pkg/github"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newApply())
}

func newApply() *cobra.Command {
	flags := struct {
		token  string
		config string
	}{}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply applies the config settings to the github repository.",
		Long:  `Apply applies the config settings to the github repository.`,
		Run: func(cmd *cobra.Command, args []string) {
			client := github.New(flags.token)

			settings, err := github.GetSettingsFromFile(flags.config)

			if err != nil {
				log.Fatal(err)
			}

			err = client.Apply(settings)

			if err != nil {
				log.Fatal(err)
			}
		},
	}

	cmd.Flags().StringVarP(&flags.config, "config", "c", "settings.yml", "Configuration file path")
	cmd.Flags().StringVarP(&flags.token, "token", "t", "", "Github personnal token")

	return cmd
}
