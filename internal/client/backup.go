package client

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

// Backup fetchs all the items and store it in the current directory.
func Backup() error {
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
	client.SetBearerToken(cfg.BearerToken)
	if cfg.Session.Defined() {
		client.SetSession(cfg.Session)
		if err = Refresh(client, &cfg); err != nil {
			return err
		}
	}

	//
	//

	auth, err := client.GetAuthParams(cfg.Email)
	if err != nil {
		return errors.Wrap(err, "could not get auth params")
	}

	if err = backup(auth, "auth_params.json"); err != nil {
		return errors.Wrap(err, "auth_params")
	}

	//
	//

	// No sync_token and limit are setted so we get all items.
	items := libsf.NewSyncItems()
	items, err = client.SyncItems(items)
	if err != nil {
		return errors.Wrap(err, "could not get items")
	}

	err = backup(items.Retrieved, fmt.Sprintf("items_%s.json", time.Now().Format("20060102150405")))
	return errors.Wrap(err, "items")
}

func backup(v any, filename string) error {
	payload, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "could not serialize value to backup")
	}

	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "could not create backup file")
	}
	defer f.Close()

	_, err = f.Write(payload)
	if err != nil {
		return errors.Wrap(err, "could not write backuped values")
	}

	return errors.Wrap(f.Sync(), "could not backup")
}
