package cmd

import (
	"github.com/geniusmonkey/gander"
	"github.com/spf13/cobra"
	"log"
)

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "Dump the migration status for the current DB",
	PreRun: dbSetup,
	PostRun: dbClose,
	Run: func(cmd *cobra.Command, args []string) {
		if err := gander.Status(db, env.MigrationsDir); err != nil {
			log.Fatalf("failed to get status, %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}