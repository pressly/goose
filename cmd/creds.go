package cmd

import (
	"fmt"
	"github.com/apex/log"
	"github.com/geniusmonkey/gander/creds"
	"github.com/geniusmonkey/gander/env"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var credsCmd = &cobra.Command{
	Use: "creds",
}

var credsAddCmd = &cobra.Command{
	Use:   "add [env name] [username]",
	Short: "add environment password for this project",
	Args:  cobra.ExactArgs(2),
	PreRun: setupProject,
	Run: func(cmd *cobra.Command, args []string) {
		envName := args[0]
		username := args[1]

		e, err := env.Get(proj, envName)
		if err != nil {
			log.Fatal("failed to get environment")
		}

		fmt.Print("Enter Password: ")
		bytePassword, err := terminal.ReadPassword(0)
		if err != nil {
			panic(err)
		}
		password := string(bytePassword)

		c := creds.Credentials{Username: username, Password: password}
		err = creds.Save(*proj, e, c)
		if err != nil {
			log.Fatalf("failed to save creds, %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(credsCmd)

	credsCmd.AddCommand(credsAddCmd)
}
