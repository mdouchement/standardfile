package database

import (
	"time"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/codec/msgpack"
	"github.com/asdine/storm/v3/q"
	"github.com/gofrs/uuid"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/pkg/errors"
)

type strm struct {
	db *storm.DB
}

// StormCodec is the format used to store data in the database.
var StormCodec = storm.Codec(msgpack.Codec)

// StormInit initializes Storm database.
func StormInit(database string) error {
	db, err := storm.Open(database, StormCodec)
	if err != nil {
		return errors.Wrap(err, "could not get database connection")
	}

	if err := db.Init(&model.User{}); err != nil {
		return errors.Wrap(err, "could not init user index")
	}

	err = db.Init(&model.Item{})
	return errors.Wrap(err, "could not init item index")
}

// StormReIndex reindex Storm database.
func StormReIndex(database string) error {
	db, err := storm.Open(database, StormCodec)
	if err != nil {
		return errors.Wrap(err, "could not get database connection")
	}

	if err := db.ReIndex(&model.User{}); err != nil {
		return errors.Wrap(err, "could not ReIndex users")
	}

	err = db.ReIndex(&model.Item{})
	return errors.Wrap(err, "could not ReIndex items")
}

// StormOpen returns a new Storm database connection.
func StormOpen(database string) (Client, error) {
	db, err := storm.Open(database, StormCodec)
	if err != nil {
		return nil, errors.Wrap(err, "could not get database connection")
	}

	return &strm{
		db: db,
	}, nil
}

// Save inserts or updates the entry in database with the given model.
func (c *strm) Save(m model.Model) error {
	t := time.Now().UTC()
	m.SetUpdatedAt(t)

	if m.GetID() == "" {
		m.SetID(uuid.Must(uuid.NewV4()).String())
		m.SetCreatedAt(t)
	}

	return errors.Wrap(c.db.Save(m), "could not save the model")
}

// Delete deletes the entry in database with the given model.
func (c *strm) Delete(m model.Model) error {
	return errors.Wrap(c.db.DeleteStruct(m), "could not delete the model")
}

// Close the database.
func (c *strm) Close() error {
	return c.db.Close()
}

// IsNotFound returns true if err is nil or a not found error.
func (c *strm) IsNotFound(err error) bool {
	return errors.Cause(err) == storm.ErrNotFound
}

// FindUser returns the user for the given id (UUID).
func (c *strm) FindUser(id string) (*model.User, error) {
	var user model.User
	if err := c.db.One("ID", id, &user); err != nil {
		return nil, errors.Wrap(err, "find user by id")
	}
	return &user, nil
}

// FindUserByMail returns the user for the given email.
func (c *strm) FindUserByMail(email string) (*model.User, error) {
	var user model.User
	if err := c.db.One("Email", email, &user); err != nil {
		return nil, errors.Wrap(err, "find user by mail")
	}
	return &user, nil
}

// FindSession returns the session for the given id (UUID).
func (c *strm) FindSession(id string) (*model.Session, error) {
	var session model.Session
	if err := c.db.One("ID", id, &session); err != nil {
		return nil, errors.Wrap(err, "find session by id")
	}
	return &session, nil
}

// FindSessionsByUserID returns all sessions for the given id and user id.
func (c *strm) FindSessionByUserID(id, userID string) (*model.Session, error) {
	var session model.Session
	err := c.db.Select(q.Eq("ID", id), q.Eq("UserID", userID)).First(&session)
	if err != nil {
		return nil, errors.Wrap(err, "find session by id and user id")
	}
	return &session, nil
}

// FindSessionByAccessToken returns the session for the given access token.
func (c *strm) FindSessionByAccessToken(token string) (*model.Session, error) {
	var session model.Session
	if err := c.db.One("AccessToken", token, &session); err != nil {
		return nil, errors.Wrap(err, "find session by access token")
	}
	return &session, nil
}

// FindSessionByTokens returns the session for the given access and refresh token.
func (c *strm) FindSessionByTokens(access, refresh string) (*model.Session, error) {
	var session model.Session
	err := c.db.Select(q.Eq("AccessToken", access), q.Eq("RefreshToken", refresh)).First(&session)
	if err != nil {
		return nil, errors.Wrap(err, "find session by tokens")
	}
	return &session, nil
}

// FindSessionsByUserID returns all the sessions for the given user id.
func (c *strm) FindSessionsByUserID(userID string) ([]*model.Session, error) {
	sessions := make([]*model.Session, 0)
	err := c.db.Select(q.Eq("UserID", userID)).OrderBy("CreatedAt").Find(&sessions)
	if err != nil && !c.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not find sessions by user id")
	}
	return sessions, nil
}

// FindActiveSessionsByUserID returns all active sessions for the given user id.
func (c *strm) FindActiveSessionsByUserID(userID string) ([]*model.Session, error) {
	sessions := make([]*model.Session, 0)
	err := c.db.Select(q.Eq("UserID", userID), q.Gt("ExpireAt", time.Now())).OrderBy("CreatedAt").Find(&sessions)
	if err != nil && !c.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not find sessions by user id")
	}
	return sessions, nil
}

// FindItem returns the item for the given id (UUID).
func (c *strm) FindItem(id string) (*model.Item, error) {
	var item model.Item
	if err := c.db.One("ID", id, &item); err != nil {
		return nil, errors.Wrap(err, "could not find item")
	}
	return &item, nil
}

// FindItemByUserID returns the item for the given id and user id (UUID).
func (c *strm) FindItemByUserID(id, userID string) (*model.Item, error) {
	var item model.Item
	err := c.db.Select(q.Eq("ID", id), q.Eq("UserID", userID)).First(&item)
	return &item, errors.Wrap(err, "could not find item by user id")
}

// FindItemsByParams returns all the matching records for the given parameters.
// It also returns a boolean to true if there is more items than the given limit.
// limit equals to 0 means all items.
func (c *strm) FindItemsByParams(userID, contentType string, updated time.Time, strictTime, noDeleted bool, limit int) ([]*model.Item, bool, error) {
	query := []q.Matcher{q.Eq("UserID", userID)}

	if !updated.IsZero() {
		if strictTime {
			query = append(query, q.Gt("UpdatedAt", updated))
		} else {
			query = append(query, q.Gte("UpdatedAt", updated))
		}
	}

	if contentType != "" {
		query = append(query, q.Eq("ContentType", contentType))
	}

	if noDeleted {
		query = append(query, q.Eq("Deleted", false))
	}

	items := make([]*model.Item, 0)
	stmt := c.db.Select(query...).OrderBy("UpdatedAt").Reverse()
	if limit > 0 {
		stmt = stmt.Limit(limit + 1)
	}
	err := stmt.Find(&items)
	if err != nil && !c.IsNotFound(err) {
		return nil, false, errors.Wrap(err, "could not find items")
	}

	var overLimit bool
	if limit != 0 && len(items) > limit {
		items = items[:limit]
		overLimit = true
	}

	return items, overLimit, nil
}

// FindItemsForIntegrityCheck returns valid items for computing data signature forthe given user.
func (c *strm) FindItemsForIntegrityCheck(userID string) ([]*model.Item, error) {
	items := make([]*model.Item, 0)
	err := c.db.Select(q.Eq("UserID", userID), q.Eq("Deleted", false), q.Not(q.Eq("ContentType", nil))).Find(&items)
	if err != nil && !c.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not find items")
	}
	return items, nil
}

// DeleteItem deletes the item matching the given parameters.
func (c *strm) DeleteItem(id, userID string) error {
	err := c.db.Select(q.Eq("ID", id), q.Eq("UserID", userID)).Delete(&model.Item{})
	return errors.Wrap(err, "could not delete item")
}
