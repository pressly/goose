package cmd

import (
	"github.com/geniusmonkey/gander/driver"
	"github.com/geniusmonkey/gander/env"
	"github.com/spf13/cobra"
	"log"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "maintain the environment configurations",
}

var envAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "add a new env variable",
	Args:  cobra.ExactArgs(1),
	PreRun: setupProject,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		e, err := driver.DefaultEnv(proj.Driver)
		if err != nil {
			log.Fatalf("failed to find driver, %v", e)
		}

		if err := env.Add(*proj, name, e); err != nil {
			log.Fatalf("failed to add env %v", e)
		}
	},
}

var envRmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove a environment",
	Args:  cobra.ExactArgs(1),
	PreRun: setup,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := env.Remove(*proj, name); err != nil {
			log.Fatalf("failed to add env %v", name)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.AddCommand(envAddCmd)
	envCmd.AddCommand(envRmCmd)
}
