package database

import (
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/asdine/storm/q"
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

// FindItemsByParams returns all the matching records for the given parameters.
// It also returns a boolean to true if there is more items than the given limit.
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
	err := c.db.Select(query...).OrderBy("UpdatedAt").Reverse().Limit(limit + 1).Find(&items)
	if err != nil && !c.IsNotFound(err) {
		return nil, false, errors.Wrap(err, "could not find items")
	}

	var overLimit bool
	if len(items) > limit {
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
