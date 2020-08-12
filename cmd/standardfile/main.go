package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/server"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const dbname = "standardfile.db"

var (
	version  = "dev"
	revision = "none"
	date     = "unknown"

	binding string
	port    string
	noreg   bool
)

func main() {
	c := &cobra.Command{
		Use:     "standardfile",
		Short:   "Standard File server for StandardNotes",
		Version: fmt.Sprintf("%s - build %.7s @ %s", version, revision, date),
		Args:    cobra.ExactArgs(0),
	}
	c.AddCommand(initCmd)
	c.AddCommand(reindexCmd)

	serverCmd.Flags().StringVarP(&binding, "binding", "b", "0.0.0.0", "Server's binding")
	serverCmd.Flags().StringVarP(&port, "port", "p", "5000", "Server's port")
	serverCmd.Flags().BoolVarP(&noreg, "noreg", "", false, "Disable registration")
	c.AddCommand(serverCmd)

	if err := c.Execute(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func dbnameWithEnv() string {
	p := os.Getenv("DATABASE_PATH")
	if len(p) == 0 {
		return dbname
	}
	return filepath.Join(p, dbname)
}

var (
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Init the database",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			return database.StormInit(dbnameWithEnv())
		},
	}

	//
	reindexCmd = &cobra.Command{
		Use:   "reindex",
		Short: "Reindex the database",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			return database.StormReIndex(dbnameWithEnv())
		},
	}

	//
	//
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start server",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			secret := os.Getenv("SECRET_KEY")
			if len(secret) == 0 {
				return errors.New("SECRET_KEY not found")
			}

			db, err := database.StormOpen(dbnameWithEnv())
			if err != nil {
				return errors.Wrap(err, "could not open database")
			}
			defer db.Close()

			engine := server.EchoEngine(server.IOC{
				Version:                    version,
				Database:                   db,
				NoRegistration:             noreg,
				SigningKey:                 []byte(secret),
				AccessTokenExpirationTime:  60 * 24 * time.Hour,  // TODO: must be a configurable value
				RefreshTokenExpirationTime: 365 * 24 * time.Hour, // TODO: must be a configurable value
			})

			server.PrintRoutes(engine)

			listen := fmt.Sprintf("%s:%s", binding, port)
			log.Printf("Server listening on %s", listen)
			return errors.Wrap(
				engine.Start(listen),
				"could not run server",
			)
		},
	}
)
