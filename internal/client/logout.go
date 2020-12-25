package client

import (
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

// Logout disconnects from a StandardFile server.
func Logout() error {
	cfg, err := Load()
	if err != nil {
		return errors.Wrap(err, "could not load config")
	}

	//
	//

	client, err := libsf.NewDefaultClient(cfg.Endpoint)
	if err != nil {
		return errors.Wrap(err, "could not reach StandardFile endpoint")
	}

	if !cfg.Session.Defined() {
		return errors.New("could not logout because session is not defined")
	}
	client.SetSession(cfg.Session)

	//
	//

	if err = client.Logout(); err != nil {
		return errors.Wrap(err, "could not logout")
	}

	return errors.Wrap(Remove(), "could not remive credential file")
}
