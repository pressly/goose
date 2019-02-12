package cmd

import (
	"github.com/geniusmonkey/gander"
	"github.com/spf13/cobra"
	"log"
)

var redoCmd = &cobra.Command{
	Use: "redo",
	Short: "Re-run the latest migration",
	PreRun: dbSetup,
	PostRun: dbClose,
	Run: func(cmd *cobra.Command, args []string) {
		if err := gander.Redo(db, env.MigrationsDir); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(redoCmd)
}