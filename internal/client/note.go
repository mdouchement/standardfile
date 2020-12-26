package client

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/mdouchement/standardfile/internal/client/tui"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

// Note runs the text-based StandardNotes application.
func Note() error {
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
			case error:
				err = r
			default:
				err = fmt.Errorf("%v", r)
			}
			stack := make([]byte, 4<<10)
			length := runtime.Stack(stack, true)

			tui.NewLogger().Printf("[PANIC RECOVER] %s %s\n", err, stack[:length])
		}
	}()

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
	ui, err := tui.New()
	if err != nil {
		return err
	}
	defer ui.Cleanup()

	// No sync_token and limit are setted to get all items.
	items := libsf.NewSyncItems()
	items, err = client.SyncItems(items)
	if err != nil {
		return errors.Wrap(err, "could not get items")
	}

	synchronizer := initSynchronizer(client, cfg, ui)

	// Append `SN|ItemsKey` to the KeyChain.
	for _, item := range items.Retrieved {
		if item.ContentType != libsf.ContentTypeItemsKey {
			continue
		}

		err := item.Unseal(&cfg.KeyChain)
		if err != nil {
			return errors.Wrap(err, "could not unseal item SN|ItemsKey")
		}
	}

	for _, item := range items.Retrieved {
		switch item.ContentType {
		case libsf.ContentTypeUserPreferences:
			err := item.Unseal(&cfg.KeyChain)
			if err != nil {
				return errors.Wrap(err, "could not unseal item SN|UserPreferences")
			}

			if err = item.Note.ParseRaw(); err != nil {
				return errors.Wrap(err, "could not parse note metadata")
			}

			ui.SortBy = item.Note.GetSortingField()
		case libsf.ContentTypeNote:
			err := item.Unseal(&cfg.KeyChain)
			if err != nil {
				return errors.Wrap(err, "could not unseal item Note")
			}

			if err = item.Note.ParseRaw(); err != nil {
				return errors.Wrap(err, "could not parse note metadata")
			}

			ui.Register(tui.NewItem(item, synchronizer))
		}
	}
	ui.SortItems()

	ui.Run()
	return nil
}

func initSynchronizer(client libsf.Client, cfg Config, ui *tui.TUI) func(item *libsf.Item) *time.Time {
	var mu sync.Mutex

	return func(item *libsf.Item) *time.Time {
		mu.Lock()
		defer mu.Unlock()

		item.Note.SetUpdatedAtNow()
		item.Note.SaveRaw()

		err := item.Seal(&cfg.KeyChain)
		if err != nil {
			ui.DisplayStatus(errors.Wrap(err, "could not seal item").Error())
			return item.UpdatedAt
		}

		items := libsf.NewSyncItems()
		items.Items = append(items.Items, item)
		items, err = client.SyncItems(items)
		if err != nil {
			ui.DisplayStatus(errors.Wrap(err, "could not get items").Error())
			return item.UpdatedAt
		}
		if len(items.Conflicts) > 0 {
			// Won't be addressed until we want several clients to run on the same account.
			// The list refreshing is done by restarting the application.
			panic("TODO: update the item proprely (item conflicts)")
		}
		ui.DisplayStatus("saved")
		ui.SortItems() // Based on local updates. No resync with the remote server is done (single client usage)

		return items.Saved[0].UpdatedAt
	}
}
