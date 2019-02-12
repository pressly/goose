package cmd

import (
	"github.com/geniusmonkey/goose"
	"github.com/spf13/cobra"
	"log"
)

var migType string

var createCmd = &cobra.Command{
	Use: "create NAME",
	Short: "Creates new migration file with the current timestamp",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := goose.Create(db, env.MigrationsDir, args[0], migType); err != nil {
			log.Fatalf("failed to create migration, %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&migType, "type", "t", "sql", "type of migration to generate 'sql' or 'go'")
}