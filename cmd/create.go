package cmd

import (
	"github.com/apex/log"
	"github.com/geniusmonkey/gander/migration"
	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
)

var migType string

var createCmd = &cobra.Command{
	Use: "create NAME",
	Short: "Creates new migration file with the current timestamp",
	Args: cobra.ExactArgs(1),
	PreRun: setupProject,
	Run: func(cmd *cobra.Command, args []string) {
		dir := proj.MigrationDir()

		if err := migration.Create(dir, slug.Make(args[0]), migType); err != nil {
			log.Fatalf("failed to create migration, %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&migType, "type", "t", "sql", "type of migration to generate 'sql' or 'go'")
}
