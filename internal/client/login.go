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

	cfg.KeyChain = *auth.SymmetricKeyPair(string(password))

	err = client.Login(auth.Email(), cfg.KeyChain.Password)
	if err != nil {
		return errors.Wrap(err, "could not login")
	}
	cfg.BearerToken = client.BearerToken() // JWT or access token
	cfg.Session = client.Session()         // Can be empty if a JWT is used

	cfg.KeyChain.Password = "" // Bearer is used instead
	return Save(cfg)
}
