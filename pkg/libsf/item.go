package libsf

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

const (
	// ContentTypeUserPreferences for items that holds user's preferences.
	ContentTypeUserPreferences = "SN|UserPreferences"
	// ContentTypePrivileges for items that holds note's privileges.
	ContentTypePrivileges = "SN|Privileges"
	// ContentTypeComponent are items that describes an editor extension.
	ContentTypeComponent = "SN|Component"
	// ContentTypeNote are the items with user's written data.
	ContentTypeNote = "Note"
)

type (
	// A SyncItems is used when a client want to sync items.
	SyncItems struct {
		// Common fields
		ComputeIntegrity bool   `json:"compute_integrity"`
		Limit            int    `json:"limit"`
		SyncToken        string `json:"sync_token"`
		CursorToken      string `json:"cursor_token"`
		ContentType      string `json:"content_type"` // optional, only return items of these type if present

		// Fields used for request
		Items []*Item `json:"items"`

		// Fields used in response
		Retrieved []*Item        `json:"retrieved_items"`
		Saved     []*Item        `json:"saved_items"`
		Unsaved   []*UnsavedItem `json:"unsaved"`
	}

	// An Item holds all the data created by end user.
	Item struct {
		ID        string     `json:"uuid"`
		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`

		UserID           string `json:"user_uuid"`
		Content          string `json:"content"`
		ContentType      string `json:"content_type"`
		EncryptedItemKey string `json:"enc_item_key"`
		Deleted          bool   `json:"deleted"`

		// Internal
		AuthParams Auth
		Note       *Note `json:"-"`

		key     vault
		content vault
	}

	// An UnsavedItem is an object containing an item that has not been saved.
	UnsavedItem struct {
		Item  Item `json:"item"`
		Error struct {
			Message string `json:"message"`
			Tag     string `json:"tag"`
		} `json:"error"`
	}
)

// NewSyncItems returns an empty SyncItems with initilized defaults.
func NewSyncItems() SyncItems {
	return SyncItems{
		Items:     []*Item{},
		Retrieved: []*Item{},
		Saved:     []*Item{},
		Unsaved:   []*UnsavedItem{},
	}
}

// Seal encrypts Note to item's Content.
func (i *Item) Seal(mk, ak string) error {
	//
	// Key
	//

	ik, err := GenerateItemEncryptionKey()
	if err != nil {
		return errors.Wrap(err, "could not generate encryption key")
	}
	i.key = vault{
		version: i.AuthParams.Version(),
		uuid:    i.ID,
		params:  i.AuthParams,
	}

	err = i.key.seal([]byte(ik), mk, ak)
	if err != nil {
		return errors.Wrap(err, "EncryptedItemKey")
	}

	i.EncryptedItemKey, err = serialize(i.key)
	if err != nil {
		return errors.Wrap(err, "EncryptedItemKey")
	}

	//
	// Content
	//

	// Split item key in encryption key and auth key
	ek := ik[:len(ik)/2]
	ak = ik[len(ik)/2:]

	note, err := json.Marshal(i.Note)
	if err != nil {
		return errors.Wrap(err, "could not serialize note")
	}

	i.content = vault{
		version: i.AuthParams.Version(),
		uuid:    i.ID,
		params:  i.AuthParams,
	}

	err = i.content.seal(note, ek, ak)
	if err != nil {
		return errors.Wrap(err, "Content")
	}

	i.Content, err = serialize(i.content)
	return errors.Wrap(err, "Content")
}

// Unseal decrypts the item's Content into Note.
func (i *Item) Unseal(mk, ak string) error {
	//
	// Key
	//

	if i.EncryptedItemKey == "" {
		return errors.New("missing item encryption key")
	}

	v, err := parse(i.EncryptedItemKey, i.ID)
	if err != nil {
		return errors.Wrap(err, "EncryptedItemKey")
	}
	i.key = v

	ik, err := i.key.unseal(mk, ak)
	if err != nil {
		return errors.Wrap(err, "EncryptedItemKey")
	}

	//
	// Content
	//

	v, err = parse(i.Content, i.ID)
	if err != nil {
		return errors.Wrap(err, "Content")
	}
	i.content = v

	// Split item key in encryption key and auth key
	ek := string(ik[:len(ik)/2])
	ak = string(ik[len(ik)/2:])

	note, err := i.content.unseal(ek, ak)
	if err != nil {
		return errors.Wrap(err, "Content")
	}

	i.Note = new(Note)
	err = json.Unmarshal(note, i.Note)
	return errors.Wrap(err, "could not parse note")
}
