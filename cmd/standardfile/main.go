package main

import (
	"fmt"
	"hash"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/server"
	"github.com/muesli/coral"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/hkdf"
)

const dbname = "standardfile.db"

var (
	version  = "dev"
	revision = "none"
	date     = "unknown"

	cfg string
)

func main() {
	c := &coral.Command{
		Use:     "standardfile",
		Short:   "Standard File server for StandardNotes",
		Version: fmt.Sprintf("%s - build %.7s @ %s - %s", version, revision, date, runtime.Version()),
		Args:    coral.ExactArgs(0),
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

func kdf(l int, k []byte) []byte {
	nhash := func() hash.Hash {
		h, err := blake2b.New256(nil)
		if err != nil {
			panic(err)
		}
		return h
	}

	payload := make([]byte, l)

	kdf := hkdf.New(nhash, k, nil, nil)
	_, err := io.ReadFull(kdf, payload)
	if err != nil {
		panic(err)
	}

	return payload
}

var (
	initCmd = &coral.Command{
		Use:   "init",
		Short: "Init the database",
		Args:  coral.ExactArgs(0),
		RunE: func(_ *coral.Command, _ []string) error {
			konf := koanf.New(".")
			if err := konf.Load(file.Provider(cfg), yaml.Parser()); err != nil {
				return err
			}

			return database.StormInit(dbnameWithPath(konf.String("database_path")))
		},
	}

	//
	reindexCmd = &coral.Command{
		Use:   "reindex",
		Short: "Reindex the database",
		Args:  coral.ExactArgs(0),
		RunE: func(_ *coral.Command, _ []string) error {
			konf := koanf.New(".")
			if err := konf.Load(file.Provider(cfg), yaml.Parser()); err != nil {
				return err
			}

			return database.StormReIndex(dbnameWithPath(konf.String("database_path")))
		},
	}

	//
	//
	serverCmd = &coral.Command{
		Use:   "server",
		Short: "Start server",
		Args:  coral.ExactArgs(0),
		RunE: func(_ *coral.Command, _ []string) error {
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
				SessionSecret:              kdf(32, konf.MustBytes("session.secret")),
				AccessTokenExpirationTime:  konf.MustDuration("session.access_token_ttl"),
				RefreshTokenExpirationTime: konf.MustDuration("session.refresh_token_ttl"),
			})
			server.PrintRoutes(engine)

			address := konf.String("address")
			message := "could not run server"
			log.Printf("Server listening on %s\n", address)
			parts := strings.Split(address, ":")
			if len(parts) == 2 && parts[0] == "unix" {
				socketFile := parts[1]
				if _, err := os.Stat(socketFile); err == nil {
					log.Printf("Removing existing %s\n", socketFile)
					os.Remove(socketFile)
				}
				defer os.Remove(socketFile)
				listener, err := net.Listen(parts[0], socketFile)
				if err != nil {
					return err
				}
				return errors.Wrap(engine.Server.Serve(listener), message)
			}
			return errors.Wrap(engine.Start(address), message)
		},
	}
)
