package database

import (
	"time"

	"github.com/mdouchement/standardfile/internal/model"
)

type (
	// A Client can interacts with the database.
	Client interface {
		// Save inserts or updates the entry in database with the given model.
		Save(m model.Model) error
		// Close the database.
		Close() error
		// IsNotFound returns true if err is nil or a not found error.
		IsNotFound(err error) bool

		UserInteraction
		ItemInteraction
	}

	// An UserInteraction defines all the methods used to interact with a user record.
	UserInteraction interface {
		// FindUser returns the user for the given id (UUID).
		FindUser(id string) (*model.User, error)
		// FindUserByMail returns the user for the given email.
		FindUserByMail(email string) (*model.User, error)
	}

	// An ItemInteraction defines all the methods used to interact with a item record(s).
	ItemInteraction interface {
		// FindItem returns the item for the given id (UUID).
		FindItem(id string) (*model.Item, error)
		// FindItemByUserID returns the item for the given id and user id (UUID).
		FindItemByUserID(id, userID string) (*model.Item, error)
		// FindItemsByParams returns all the matching records for the given parameters.
		// It also returns a boolean to true if there is more items than the given limit.
		FindItemsByParams(userID, contentType string, updated time.Time, strictTime, filterDeleted bool, limit int) ([]*model.Item, bool, error)
		// FindItemsForIntegrityCheck returns valid items for computing data signature forthe given user.
		FindItemsForIntegrityCheck(userID string) ([]*model.Item, error)
		// DeleteItem deletes the item matching the given parameters.
		DeleteItem(id, userID string) error
	}
)
