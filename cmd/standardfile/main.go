package main

import (
	"context"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/labstack/echo/v5"
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

			var subscription, features []byte
			if konf.String("subscription_file") != "" {
				subscription, err = os.ReadFile(konf.String("subscription_file"))
				if err != nil {
					return errors.Wrap(err, "could not read subscription_file")
				}

				features, err = os.ReadFile(konf.String("features_file"))
				if err != nil {
					return errors.Wrap(err, "could not read features_file")
				}
			}

			engine := server.EchoEngine(server.Controller{
				Version:                    version,
				Database:                   db,
				NoRegistration:             konf.Bool("no_registration"),
				ShowRealVersion:            konf.Bool("show_real_version"),
				SubscriptionPayload:        subscription,
				FeaturesPayload:            features,
				AllowOrigins:               konf.MustStrings("cors.allow_origins"),
				AllowMethods:               konf.MustStrings("cors.allow_methods"),
				SigningKey:                 configSecretKey,
				SessionSecret:              kdf(32, configSessionSecret),
				AccessTokenExpirationTime:  konf.MustDuration("session.access_token_ttl"),
				RefreshTokenExpirationTime: konf.MustDuration("session.refresh_token_ttl"),
			})
			server.PrintRoutes(engine)

			address, err := parseAddress(konf.String("address"))
			if err != nil {
				return errors.Wrap(err, "invalid listening address")
			}

			sc := echo.StartConfig{
				Address:         address.Host, // default to TCP used by HTTP
				GracefulTimeout: 10 * time.Second,
			}

			if address.Scheme == "unix" {
				socket := address.Path
				if _, err := os.Stat(socket); err == nil {
					log.Printf("Removing existing %s\n", socket)
					os.Remove(socket)
				}
				defer os.Remove(socket)

				listener, err := net.Listen("unix", socket)
				if err != nil {
					return err
				}

				if socketMode := konf.Int("socket_mode"); socketMode != 0 {
					mode := fs.FileMode(socketMode)
					if err := os.Chmod(socket, mode); err != nil {
						return errors.Wrap(err, fmt.Sprintf("chmod %s %#o", socket, mode))
					}
				}

				sc.Address = socket
				sc.Listener = listener
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			log.Printf("Server listening on %s\n", sc.Address)
			return errors.Wrap(sc.Start(ctx, engine), "could not run server")
		},
	}
)

func parseAddress(addr string) (*url.URL, error) {
	// Case of `:5000' listen address is provided
	if regexp.MustCompile(`^:\d+$`).MatchString(addr) {
		addr = fmt.Sprintf("tcp://0.0.0.0%s", addr)
	}

	// Case of `localhost:5000' listen address is provided
	if !strings.Contains(addr, "://") {
		addr = fmt.Sprintf("tcp://%s", addr)
	}

	return url.Parse(addr)
}
