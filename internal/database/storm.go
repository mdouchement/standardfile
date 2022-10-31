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

func (c *strm) Save(m model.Model) error {
	t := time.Now().UTC()
	m.SetUpdatedAt(t)

	if m.GetID() == "" {
		m.SetID(uuid.Must(uuid.NewV4()).String())
		m.SetCreatedAt(t)
	}

	return errors.Wrap(c.db.Save(m), "could not save the model")
}

func (c *strm) Delete(m model.Model) error {
	return errors.Wrap(c.db.DeleteStruct(m), "could not delete the model")
}

func (c *strm) Close() error {
	return c.db.Close()
}

func (c *strm) IsNotFound(err error) bool {
	return errors.Cause(err) == storm.ErrNotFound
}

func (c *strm) IsAlreadyExists(err error) bool {
	return errors.Cause(err) == storm.ErrAlreadyExists
}

func (c *strm) FindUser(id string) (*model.User, error) {
	var user model.User
	if err := c.db.One("ID", id, &user); err != nil {
		return nil, errors.Wrap(err, "find user by id")
	}
	return &user, nil
}

func (c *strm) FindUserByMail(email string) (*model.User, error) {
	var user model.User
	if err := c.db.One("Email", email, &user); err != nil {
		return nil, errors.Wrap(err, "find user by mail")
	}
	return &user, nil
}

func (c *strm) FindSession(id string) (*model.Session, error) {
	var session model.Session
	if err := c.db.One("ID", id, &session); err != nil {
		return nil, errors.Wrap(err, "find session by id")
	}
	return &session, nil
}

func (c *strm) FindSessionByUserID(id, userID string) (*model.Session, error) {
	var session model.Session
	err := c.db.Select(q.Eq("ID", id), q.Eq("UserID", userID)).First(&session)
	if err != nil {
		return nil, errors.Wrap(err, "find session by id and user id")
	}
	return &session, nil
}

func (c *strm) FindSessionByAccessToken(id, token string) (*model.Session, error) {
	var session model.Session
	err := c.db.Select(q.Eq("ID", id), q.Eq("AccessToken", token)).First(&session)
	if err != nil {
		return nil, errors.Wrap(err, "find session by access token")
	}
	return &session, nil
}

func (c *strm) FindSessionByTokens(id, access, refresh string) (*model.Session, error) {
	var session model.Session
	err := c.db.Select(q.Eq("ID", id), q.Eq("AccessToken", access), q.Eq("RefreshToken", refresh)).First(&session)
	if err != nil {
		return nil, errors.Wrap(err, "could not find session by tokens")
	}
	return &session, nil
}

func (c *strm) FindSessionsByUserID(userID string) ([]*model.Session, error) {
	sessions := make([]*model.Session, 0)
	err := c.db.Select(q.Eq("UserID", userID)).OrderBy("CreatedAt").Find(&sessions)
	if err != nil && !c.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not find sessions by user id")
	}
	return sessions, nil
}

func (c *strm) FindActiveSessionsByUserID(userID string) ([]*model.Session, error) {
	sessions := make([]*model.Session, 0)
	err := c.db.Select(q.Eq("UserID", userID), q.Gt("ExpireAt", time.Now())).OrderBy("CreatedAt").Find(&sessions)
	if err != nil && !c.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not find sessions by user id")
	}
	return sessions, nil
}

func (c *strm) FindItem(id string) (*model.Item, error) {
	var item model.Item
	if err := c.db.One("ID", id, &item); err != nil {
		return nil, errors.Wrap(err, "could not find item")
	}
	return &item, nil
}

func (c *strm) FindItemByUserID(id, userID string) (*model.Item, error) {
	var item model.Item
	err := c.db.Select(q.Eq("ID", id), q.Eq("UserID", userID)).First(&item)
	return &item, errors.Wrap(err, "could not find item by user id")
}

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

func (c *strm) FindItemsForIntegrityCheck(userID string) ([]*model.Item, error) {
	items := make([]*model.Item, 0)
	err := c.db.Select(q.Eq("UserID", userID), q.Eq("Deleted", false), q.Not(q.Eq("ContentType", nil))).Find(&items)
	if err != nil && !c.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not find items")
	}
	return items, nil
}

func (c *strm) DeleteItem(id, userID string) error {
	err := c.db.Select(q.Eq("ID", id), q.Eq("UserID", userID)).Delete(&model.Item{})
	return errors.Wrap(err, "could not delete item")
}

func (c *strm) FindPKCE(codeChallenge string) (*model.PKCE, error) {
	var pkce model.PKCE
	err := c.db.Select(q.Eq("CodeChallenge", codeChallenge)).First(&pkce)
	if err != nil {
		return nil, errors.Wrap(err, "could not find pkce")
	}
	return &pkce, nil
}

func (c *strm) RevokeExpiredChallenges() error {
	err := c.db.Select(q.Lte("ExpireAt", time.Now().UTC())).Delete(&model.PKCE{})
	if c.IsNotFound(err) {
		return nil
	}
	return errors.Wrap(err, "could not delete expired challenges")
}

func (c *strm) RemovePKCE(codeChallenge string) error {
	err := c.db.Select(q.Eq("CodeChallenge", codeChallenge)).Delete(&model.PKCE{})
	return errors.Wrap(err, "Could not delete challenge")
}
