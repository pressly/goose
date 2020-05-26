package cmd

import (
	"fmt"
	"github.com/apex/log"
	"github.com/geniusmonkey/gander/creds"
	"github.com/geniusmonkey/gander/db"
	"github.com/geniusmonkey/gander/env"
	"github.com/geniusmonkey/gander/project"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"os"
)

//var cfgFile string
var (
	proj       *project.Project
	projectDir string
	//environment *env.Environment
)

//var envName string
//var db *sql.DB

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gander",
	Short: "CLI for running SQL migrations",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	//c.Version = Version
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectDir, "project", "p", "./", "location of the project")
}

func setupProject(cmd *cobra.Command, args []string) {
	var err error
	//var dir string
	//if d, err := os.Getwd(); err != nil {
	//	log.Fatal("failed to get project directory")
	//} else {
	//	dir = d
	//}

	proj, err = project.Get(projectDir)
	if err != nil && err != project.IsNotExists {
		log.Fatalf("error while trying to load project directory: %v", err)
	}
}

func setup(cmd *cobra.Command, args []string) {
	setupProject(cmd, args)

	if proj == nil {
		log.Fatalf("not a gander project")
	}

	var envName = proj.DefaultEnv
	if len(args) == 1 {
		envName = args[0]
	}

	environment, err := env.Get(proj, envName)
	if err != nil {
		log.Fatalf("failed to get environment: %v", err)
	}

	cred, err := creds.Get(*proj, environment)
	if err != nil {
		log.Fatalf("credentials not found for environment %v, \n\tuse \"gander creds add\"", environment.Name)
	}

	db.Setup(*proj, environment, cred)
}

func tearDown(cmd *cobra.Command, args []string) {
	db.Close()
}
