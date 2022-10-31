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
		// Delete deletes the entry in database with the given model.
		Delete(m model.Model) error
		// Close the database.
		Close() error
		// IsNotFound returns true if err is a not found error.
		IsNotFound(err error) bool
		// IsAlreadyExists returns true if err is a not found error.
		IsAlreadyExists(err error) bool

		UserInteraction
		SessionInteraction
		ItemInteraction
		PKCEInteraction
	}

	// An UserInteraction defines all the methods used to interact with a user record.
	UserInteraction interface {
		// FindUser returns the user for the given id (UUID).
		FindUser(id string) (*model.User, error)
		// FindUserByMail returns the user for the given email.
		FindUserByMail(email string) (*model.User, error)
	}

	// An SessionInteraction defines all the methods used to interact with a session record.
	SessionInteraction interface {
		// FindSession returns the session for the given id (UUID).
		FindSession(id string) (*model.Session, error)
		// FindSessionsByUserID returns all sessions for the given id and user id.
		FindSessionByUserID(id, userID string) (*model.Session, error)
		// FindActiveSessionsByUserID returns all active sessions for the given user id.
		FindActiveSessionsByUserID(userID string) ([]*model.Session, error)
		// FindSessionsByUserID returns all sessions for the given user id.
		FindSessionsByUserID(userID string) ([]*model.Session, error)
		// FindSessionByAccessToken returns the session for the given id and access token.
		FindSessionByAccessToken(id, token string) (*model.Session, error)
		// FindSessionByTokens returns the session for the given id, access and refresh token.
		FindSessionByTokens(id, access, refresh string) (*model.Session, error)
	}

	// An ItemInteraction defines all the methods used to interact with a item record(s).
	ItemInteraction interface {
		// FindItem returns the item for the given id (UUID).
		FindItem(id string) (*model.Item, error)
		// FindItemByUserID returns the item for the given id and user id (UUID).
		FindItemByUserID(id, userID string) (*model.Item, error)
		// FindItemsByParams returns all the matching records for the given parameters.
		// It also returns a boolean to true if there is more items than the given limit.
		// limit equals to 0 means all items.
		FindItemsByParams(userID, contentType string, updated time.Time, strictTime, filterDeleted bool, limit int) ([]*model.Item, bool, error)
		// FindItemsForIntegrityCheck returns valid items for computing data signature forthe given user.
		FindItemsForIntegrityCheck(userID string) ([]*model.Item, error)
		// DeleteItem deletes the item matching the given parameters.
		DeleteItem(id, userID string) error
	}

	// A PKCEInteraction defines all the methods used to interact with PKCE mechanism.
	PKCEInteraction interface {
		// FindPKCE returns the item for the given code.
		FindPKCE(codeChallenge string) (*model.PKCE, error)
		// RemovePKCE removes from database the given challenge code.
		RemovePKCE(codeChallenge string) error
		// RevokeExpiredChallenges removes from database all old challenge codes.
		RevokeExpiredChallenges() error
	}
)
