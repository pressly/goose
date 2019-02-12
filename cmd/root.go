package cmd

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/geniusmonkey/gander"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
)

var cfgFile string
var env = &gander.Environment{}
var envName string
var db *sql.DB

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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file location (default dbconf.toml)")
	rootCmd.PersistentFlags().StringVarP(&envName, "env", "e", "development", "name of the environment to use")

	rootCmd.PersistentFlags().StringVar(&env.Dsn, "dsn", "", "dataSourceName to connect to the server")
	rootCmd.PersistentFlags().StringVar(&env.Driver, "driver", "", "name of the database driver")
	rootCmd.PersistentFlags().StringVar(&env.MigrationsDir, "dir", "./migrations", "directory containing the migration files")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		if _, err := os.Stat("dbconf.toml"); os.IsNotExist(err) {
			return
		} else if cfgFile == "" {
			cfgFile = "dbconf.toml"
		}

	}

	file, err := os.Open(cfgFile)
	info, _ := file.Stat()
	if os.IsNotExist(err) {
		log.Fatalf("no config file found at %s", cfgFile)
	}

	c := gander.Config{}
	if _, err := toml.DecodeReader(file, &c); err != nil {
		log.Fatalf("failed to read file %s", err)
	}

	if e, ok := c.Environments[envName]; !ok {
		log.Fatalf("failed to environment named %s in %s", envName, info.Name())
	} else {
		env = &e
	}

	env.MigrationsDir = filepath.Join(filepath.Dir(cfgFile), env.MigrationsDir)
}

func dbSetup(cmd *cobra.Command, args []string) {
	var err error
	if err := gander.SetDialect(env.Driver); err != nil {
		log.Fatalf("failed to set dialect %s, %v", env.Driver, err)
	}

	switch env.Driver {
	case "redshift", "cockroach":
		env.Driver = "postgres"
	case "tidb":
		env.Driver = "mysql"
	}

	db, err = sql.Open(env.Driver, env.Dsn)
	if err != nil {
		log.Fatalf("failed to open connection %s", err)
	}
}

func dbClose(cmd *cobra.Command, args []string) {
	_ = db.Close()
}
