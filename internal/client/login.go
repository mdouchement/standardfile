package client

import (
	"github.com/chzyer/readline"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

// Login connects to a StandardFile server.
func Login() error {
	cfg := Config{}

	endpoint, err := readline.Line("Endpoint: ")
	if err != nil {
		return errors.Wrap(err, "could not read endpoint from stdin")
	}
	cfg.Endpoint = endpoint

	client, err := libsf.NewDefaultClient(cfg.Endpoint)
	if err != nil {
		return errors.Wrap(err, "could not reach given endpoint")
	}

	cfg.Email, err = readline.Line("Email: ")
	if err != nil {
		return errors.Wrap(err, "could not read email from stdin")
	}

	auth, err := client.GetAuthParams(cfg.Email)
	if err != nil {
		return errors.Wrap(err, "could not get auth params")
	}
	if err = auth.IntegrityCheck(); err != nil {
		return errors.Wrap(err, "invalid auth params")
	}

	password, err := readline.Password("Password: ")
	if err != nil {
		return errors.Wrap(err, "could not read password from stdin")
	}

	keychain := auth.SymmetricKeyPair(string(password))
	cfg.Mk = keychain.MasterKey
	cfg.Ak = keychain.AuthKey

	err = client.Login(auth.Email(), keychain.Password)
	if err != nil {
		return errors.Wrap(err, "could not login")
	}
	cfg.BearerToken = client.BearerToken()

	return Save(cfg)
}
