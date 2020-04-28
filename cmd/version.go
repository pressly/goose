package cmd

import (
	"github.com/geniusmonkey/gander/db"
	"github.com/geniusmonkey/gander/migration"
	"github.com/spf13/cobra"
	"log"
)

var verCmd = &cobra.Command{
	Use:   "version",
	Short: "Info the current version of the database",
	PreRun: setup,
	PostRun: tearDown,
	Run: func(cmd *cobra.Command, args []string) {
		if err := migration.Version(db.Get(), proj.MigrationDir()); err != nil {
			log.Fatalf("failed to get current db version, %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(verCmd)
}
