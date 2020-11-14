package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
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

	cfg string
)

func main() {
	c := &cobra.Command{
		Use:     "standardfile",
		Short:   "Standard File server for StandardNotes",
		Version: fmt.Sprintf("%s - build %.7s @ %s", version, revision, date),
		Args:    cobra.ExactArgs(0),
	}
	initCmd.Flags().StringVarP(&cfg, "config", "c", "", "Configuration file")
	c.AddCommand(initCmd)

	reindexCmd.Flags().StringVarP(&cfg, "config", "c", "", "Configuration file")
	c.AddCommand(reindexCmd)

	serverCmd.Flags().StringVarP(&cfg, "config", "c", "", "Configuration file")
	c.AddCommand(serverCmd)

	if err := c.Execute(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func dbnameWithPath(path string) string {
	if len(path) == 0 {
		return dbname
	}
	return filepath.Join(path, dbname)
}

var (
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Init the database",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			konf := koanf.New(".")
			if err := konf.Load(file.Provider(cfg), toml.Parser()); err != nil {
				return err
			}

			return database.StormInit(dbnameWithPath(konf.String("database_path")))
		},
	}

	//
	reindexCmd = &cobra.Command{
		Use:   "reindex",
		Short: "Reindex the database",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			konf := koanf.New(".")
			if err := konf.Load(file.Provider(cfg), toml.Parser()); err != nil {
				return err
			}

			return database.StormReIndex(dbnameWithPath(konf.String("database_path")))
		},
	}

	//
	//
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start server",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			konf := koanf.New(".")
			if err := konf.Load(file.Provider(cfg), yaml.Parser()); err != nil {
				return err
			}

			if konf.String("secret_key") == "" {
				return errors.New("secret_key not found")
			}

			if konf.String("session.secret") == "" {
				return errors.New("session secret not found")
			}

			db, err := database.StormOpen(dbnameWithPath(konf.String("database_path")))
			if err != nil {
				return errors.Wrap(err, "could not open database")
			}
			defer db.Close()

			engine := server.EchoEngine(server.Controller{
				Version:                    version,
				Database:                   db,
				NoRegistration:             konf.Bool("no_registration"),
				SigningKey:                 konf.MustBytes("secret_key"),
				AccessTokenExpirationTime:  konf.MustDuration("session.access_token_ttl"),
				RefreshTokenExpirationTime: konf.MustDuration("session.refresh_token_ttl"),
			})
			server.PrintRoutes(engine)

			log.Printf("Server listening on %s", konf.String("address"))
			return errors.Wrap(
				engine.Start(konf.String("address")),
				"could not run server",
			)
		},
	}
)
