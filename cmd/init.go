package cmd

import (
	"github.com/geniusmonkey/gander/project"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var initProj = project.Project{
}

var initCmd = &cobra.Command{
	Use: "init",
	Short: "Used to init a new gander project configuration",
	Run: func(cmd *cobra.Command, args []string) {
		var dir string
		if d, err := os.Getwd(); err != nil {
			log.Fatal("failed to get current directory")
		} else {
			dir = d
		}

		if err := project.Init(dir, initProj); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	initCmd.Flags().StringVarP(&initProj.Driver, "driver", "d", "mysql", "driver to use for project")
	initCmd.Flags().StringVarP(&initProj.Migrations, "migrations", "m", "./migrations", "directory with the migrations files")
	initCmd.Flags().StringVarP(&initProj.DefaultEnv, "defaultEnv", "e", "local", "name of the default environment")
	initCmd.Flags().StringVarP(&initProj.Name, "name", "n", "", "name of the project")
	rootCmd.AddCommand(initCmd)
}


