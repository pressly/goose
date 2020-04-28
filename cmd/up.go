package cmd

import (
	"github.com/geniusmonkey/gander/db"
	"github.com/geniusmonkey/gander/migration"
	"github.com/spf13/cobra"
	"log"
)

var upTo int64

var upCmd = &cobra.Command{
	Use: "up",
	Short: "Migrate the DB to the most recent version available",
	PreRun: setup,
	PostRun: tearDown,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if upTo == 0 {
			err = migration.Up(db.Get(), proj.MigrationDir())
		} else {
			err = migration.UpTo(db.Get(), proj.MigrationDir(), upTo)
		}

		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().Int64VarP(&upTo, "to", "t", 0, "migrate up to a specific version")
}