package cmd

import (
	"github.com/geniusmonkey/gander"
	"github.com/spf13/cobra"
	"log"
)

var verCmd = &cobra.Command{
	Use: "version",
	Short: "Print the current version of the database",
	PreRun: dbSetup,
	PostRun: dbClose,
	Run: func(cmd *cobra.Command, args []string) {
		if err := gander.Version(db, env.MigrationsDir); err != nil {
			log.Fatalf("failed to get current db version, %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(verCmd)
}