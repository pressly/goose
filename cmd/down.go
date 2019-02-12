package cmd

import (
	"github.com/geniusmonkey/goose"
	"github.com/spf13/cobra"
	"log"
)

var downTo int64

var downCmd = &cobra.Command{
	Use: "down",
	Short: "Roll back the version by 1",
	PreRun: dbSetup,
	PostRun: dbClose,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if downTo == 0 {
			err = goose.Down(db, env.MigrationsDir)
		} else {
			err = goose.DownTo(db, env.MigrationsDir, downTo)
		}

		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().Int64VarP(&downTo, "to", "t", 0, "roll back to a specific version")
}