package client

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

// Unseal decrypt your backuped notes.
func Unseal(filename string) error {
	cfg, err := Load()
	if err != nil {
		return errors.Wrap(err, "could not load config")
	}

	//
	//

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "could not load file")
	}

	sync := libsf.NewSyncItems()
	if err = json.Unmarshal(data, &sync.Retrieved); err != nil {
		return errors.Wrap(err, "could not parse backuped notes")
	}

	//
	//

	var notes []libsf.Note

	for _, rt := range sync.Retrieved {
		if rt.ContentType != libsf.ContentTypeNote {
			continue
		}

		err = rt.Unseal(cfg.Mk, cfg.Ak)
		if err != nil {
			return errors.Wrap(err, "could not unseal item")
		}

		notes = append(notes, *rt.Note)
	}

	err = backup(notes, strings.Replace(filename, "items_", "notes_", 1))
	return errors.Wrap(err, "notes")
}
