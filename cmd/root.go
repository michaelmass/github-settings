package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// VERSION of github-settings
// nolint:gochecknoglobals
var VERSION string

const (
	defaultFolderPermission = 0755
	defaultFilePermission   = 0644
)

var rootCmd = &cobra.Command{
	Use:   "github-settings",
	Short: "github-settings is a setttings configuration tool for github",
	Long:  `github-settings is a setttings configuration tool for github.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("github-settings version %s\n", VERSION)
	},
}

// Execute the cli
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
