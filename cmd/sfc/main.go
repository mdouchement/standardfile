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
	c.AddCommand(logoutCmd)
	c.AddCommand(backupCmd)
	c.AddCommand(unsealCmd)
	c.AddCommand(noteCmd)

	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to the StandardFile server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return client.Login()
		},
	}

	logoutCmd = &cobra.Command{
		Use:   "logout",
		Short: "Logout from a StandardFile server session",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return client.Logout()
		},
	}

	backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "Backup your notes",
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

	noteCmd = &cobra.Command{
		Use:   "note",
		Short: "Text-based StandardNotes application",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return client.Note()
		},
	}
)
