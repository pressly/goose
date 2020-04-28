package cmd

import (
	"github.com/geniusmonkey/gander/db"
	"github.com/geniusmonkey/gander/migration"
	"github.com/spf13/cobra"
	"log"
)

var statusCmd = &cobra.Command{
	Use: "status [env]",
	Short: "Dump the migration status for the current DB",
	Args: cobra.MaximumNArgs(1),
	PreRun: setup,
	PostRun: tearDown,
	Run: func(cmd *cobra.Command, args []string) {
		if err := migration.Status(db.Get(), proj.MigrationDir()); err != nil {
			log.Fatalf("failed to get status, %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}