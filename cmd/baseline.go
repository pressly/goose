package cmd

import (
	"github.com/geniusmonkey/gander"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

var baselineCmd = &cobra.Command{
	Use:     "baseline VERSION",
	Short:   "Baseline an existing db to a specific VERSION",
	Args:    cobra.ExactArgs(1),
	PreRun:  dbSetup,
	PostRun: dbClose,
	Run: func(cmd *cobra.Command, args []string) {
		ver, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			log.Fatalf("unable to convert version %s into number, %s", args[0], err)
		}

		if err := gander.Baseline(db, env.MigrationsDir, ver); err != nil {
			log.Fatalf("failed to create migration, %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(baselineCmd)
}
