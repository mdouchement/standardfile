package main

import (
	"fmt"
	"os"

	"github.com/mdouchement/standardfile/internal/client"
	"github.com/spf13/cobra"
)

var (
	version  = "dev"
	revision = "none"
	date     = "unknown"
)

func main() {
	c := &cobra.Command{
		Use:     "sfc",
		Short:   "Standard File client (aka StandardNotes)",
		Version: fmt.Sprintf("%s - build %.7s @ %s", version, revision, date),
		Args:    cobra.NoArgs,
	}
	c.AddCommand(loginCmd)
	c.AddCommand(backupCmd)
	c.AddCommand(unsealCmd)

	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "login to the StandardFile server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return client.Login()
		},
	}

	backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "backup your notes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return client.Backup()
		},
	}

	unsealCmd = &cobra.Command{
		Use:   "unseal FILENAME",
		Short: "Decrypt your backuped notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return client.Unseal(args[0])
		},
	}
)
