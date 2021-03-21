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
	// ContentTypeItemsKey are items used to encrypt Note items.
	ContentTypeItemsKey = "SN|ItemsKey"
	// ContentTypeNote are the items with user's written data.
	ContentTypeNote = "Note"
)

type (
	// A SyncItems is used when a client want to sync items.
	SyncItems struct {
		// Common fields
		API              string `json:"api"` // Since 20190520
		ComputeIntegrity bool   `json:"compute_integrity,omitempty"`
		Limit            int    `json:"limit,omitempty"`
		SyncToken        string `json:"sync_token,omitempty"`
		CursorToken      string `json:"cursor_token,omitempty"`
		ContentType      string `json:"content_type,omitempty"` // optional, only return items of these type if present

		// Fields used for request
		Items []*Item `json:"items,omitempty"`

		// Fields used in response
		Retrieved []*Item `json:"retrieved_items,omitempty"`
		Saved     []*Item `json:"saved_items,omitempty"`

		Unsaved   []*UnsavedItem  `json:"unsaved,omitempty"`   // Before 20190520 (Since 20161215 at least)
		Conflicts []*ConflictItem `json:"conflicts,omitempty"` // Since 20190520
	}

	// An Item holds all the data created by end user.
	Item struct {
		ID        string     `json:"uuid"`
		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`

		UserID           string `json:"user_uuid"`
		Content          string `json:"content"`
		ContentType      string `json:"content_type"`
		ItemsKeyID       string `json:"items_key_id"` // Since 20200115
		EncryptedItemKey string `json:"enc_item_key"`
		Deleted          bool   `json:"deleted"`

		// Internal
		Version    string `json:"-"`
		AuthParams Auth   `json:"-"`
		Note       *Note  `json:"-"`

		key     vault
		content vault
	}

	// An UnsavedItem is an object containing an item that has not been saved.
	// Used before API version 20190520.
	UnsavedItem struct {
		Item  Item `json:"item"`
		Error struct {
			Message string `json:"message"`
			Tag     string `json:"tag"`
		} `json:"error"`
	}

	// A ConflictItem is an object containing an item that can't be saved caused by conflicts.
	// Used since API version 20190520.
	ConflictItem struct {
		UnsavedItem Item   `json:"unsaved_item,omitempty"`
		ServerItem  Item   `json:"server_item,omitempty"`
		Type        string `json:"type"`
	}
)

// NewSyncItems returns an empty SyncItems with initilized defaults.
func NewSyncItems() SyncItems {
	return SyncItems{
		Items:     []*Item{},
		Retrieved: []*Item{},
		Saved:     []*Item{},
		Unsaved:   []*UnsavedItem{},
		Conflicts: []*ConflictItem{},
	}
}

// Seal encrypts Note to item's Content.
func (i *Item) Seal(keychain *KeyChain) error {
	//
	// Key
	//

	ik, err := keychain.GenerateItemEncryptionKey()
	if err != nil {
		return errors.Wrap(err, "could not generate encryption key")
	}

	i.key, err = create(i.Version, i.ID)
	if err != nil {
		return errors.Wrap(err, "could not create vault")
	}
	i.key.setup(i)

	err = i.key.seal(keyKeyChain(keychain, i), []byte(ik))
	if err != nil {
		return errors.Wrap(err, "EncryptedItemKey")
	}

	i.EncryptedItemKey, err = i.key.serialize()
	if err != nil {
		return errors.Wrap(err, "EncryptedItemKey")
	}

	//
	// Content
	//

	note, err := json.Marshal(i.Note)
	if err != nil {
		return errors.Wrap(err, "could not serialize note")
	}

	i.content, err = create(i.Version, i.ID)
	if err != nil {
		return errors.Wrap(err, "could not create content vault")
	}
	i.content.setup(i)

	err = i.content.seal(contentKeyChain(i.Version, ik), note)
	if err != nil {
		return errors.Wrap(err, "Content")
	}

	i.Content, err = i.content.serialize()
	return errors.Wrap(err, "Content")
}

// Unseal decrypts the item's Content into Note.
// `SN|ItemsKey` are append in the provided KeyChain.
func (i *Item) Unseal(keychain *KeyChain) error {
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
	i.key.configure(i)

	ik, err := i.key.unseal(keyKeyChain(keychain, i))
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

	payload, err := i.content.unseal(contentKeyChain(i.Version, string(ik)))
	if err != nil {
		return errors.Wrap(err, "Content")
	}

	switch i.ContentType {
	case ContentTypeItemsKey:
		v := &struct {
			ItemKeys string `json:"itemsKey"`
		}{}

		err = json.Unmarshal(payload, v)
		keychain.ItemsKey[i.ID] = v.ItemKeys
		return errors.Wrap(err, "could not parse items key")
	case ContentTypeUserPreferences:
		fallthrough
	case ContentTypeNote:
		i.Note = new(Note)
		err = json.Unmarshal(payload, i.Note)
		return errors.Wrap(err, "could not parse note")
	}

	return errors.Errorf("Unsupported unseal for %s", i.ContentType)
}
