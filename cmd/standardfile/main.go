package main

import (
	"fmt"
	"hash"
	"io"
	"io/fs"
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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
	c := &cobra.Command{
		Use:     "standardfile",
		Short:   "Standard File server for StandardNotes",
		Version: fmt.Sprintf("%s - build %.7s @ %s - %s", version, revision, date, runtime.Version()),
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

// keyFromConfig reads a key from the configuration, and if it's not present, tries to read it from a file instead
func keyFromConfig(konf *koanf.Koanf, path string) (out []byte, err error) {
	// check if the key is directly placed in the config file
	out = konf.Bytes(path)
	if len(out) > 0 {
		return out, nil
	}

	// check if the key is available as a systemd credential
	credsDir := os.Getenv("CREDENTIALS_DIRECTORY")
	if credsDir == "" {
		return nil, errors.New("not found")
	}
	filename := filepath.Join(credsDir, path)

	out, err = os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "file read")
	}

	if len(out) == 0 {
		return nil, errors.New("file empty")
	}

	return out, nil
}

var (
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Init the database",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			konf := koanf.New(".")
			if err := konf.Load(file.Provider(cfg), yaml.Parser()); err != nil {
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
			if err := konf.Load(file.Provider(cfg), yaml.Parser()); err != nil {
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

			configSecretKey, err := keyFromConfig(konf, "secret_key")
			if err != nil {
				return errors.Wrap(err, "secret key")
			}

			configSessionSecret, err := keyFromConfig(konf, "session.secret")
			if err != nil {
				return errors.Wrap(err, "session secret")
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
				ShowRealVersion:            konf.Bool("show_real_version"),
				EnableSubscription:         konf.Bool("enable_subscription"),
				FilesServerUrl:             konf.String("files_server_url"),
				SigningKey:                 configSecretKey,
				SessionSecret:              kdf(32, configSessionSecret),
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

				if socketMode := konf.Int("socket_mode"); socketMode != 0 {
					mode := fs.FileMode(socketMode)
					if err := os.Chmod(socketFile, mode); err != nil {
						return errors.Wrap(err, fmt.Sprintf("chmod %s %#o", socketFile, mode))
					}
				}

				return errors.Wrap(engine.Server.Serve(listener), message)
			}

			return errors.Wrap(engine.Start(address), message)
		},
	}
)
