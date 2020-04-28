package cmd

import (
	"github.com/geniusmonkey/gander/db"
	"github.com/geniusmonkey/gander/migration"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

var baselineCmd = &cobra.Command{
	Use:     "baseline VERSION",
	Short:   "Baseline an existing db to a specific VERSION",
	Args:    cobra.ExactArgs(1),
	PreRun: setup,
	PostRun: tearDown,
	Run: func(cmd *cobra.Command, args []string) {
		ver, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			log.Fatalf("unable to convert version %s into number, %s", args[0], err)
		}

		if err := migration.Baseline(db.Get(), proj.MigrationDir(), ver); err != nil {
			log.Fatalf("failed to create migration, %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(baselineCmd)
}
