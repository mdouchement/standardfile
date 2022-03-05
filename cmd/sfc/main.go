package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/mdouchement/standardfile/internal/client"
	"github.com/muesli/coral"
)

var (
	version  = "dev"
	revision = "none"
	date     = "unknown"
)

func main() {
	c := &coral.Command{
		Use:     "sfc",
		Short:   "Standard File client (aka StandardNotes)",
		Version: fmt.Sprintf("%s - build %.7s @ %s - %s", version, revision, date, runtime.Version()),
		Args:    coral.NoArgs,
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
	loginCmd = &coral.Command{
		Use:   "login",
		Short: "Login to the StandardFile server",
		Args:  coral.NoArgs,
		RunE: func(_ *coral.Command, args []string) error {
			return client.Login()
		},
	}

	logoutCmd = &coral.Command{
		Use:   "logout",
		Short: "Logout from a StandardFile server session",
		Args:  coral.NoArgs,
		RunE: func(_ *coral.Command, args []string) error {
			return client.Logout()
		},
	}

	backupCmd = &coral.Command{
		Use:   "backup",
		Short: "Backup your notes",
		Args:  coral.NoArgs,
		RunE: func(_ *coral.Command, args []string) error {
			return client.Backup()
		},
	}

	unsealCmd = &coral.Command{
		Use:   "unseal FILENAME",
		Short: "Decrypt your backuped notes",
		Args:  coral.ExactArgs(1),
		RunE: func(_ *coral.Command, args []string) error {
			return client.Unseal(args[0])
		},
	}

	noteCmd = &coral.Command{
		Use:   "note",
		Short: "Text-based StandardNotes application",
		Args:  coral.NoArgs,
		RunE: func(_ *coral.Command, args []string) error {
			return client.Note()
		},
	}
)
