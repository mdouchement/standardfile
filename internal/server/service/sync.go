package service

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

type (
	// A SyncParams is used when a client want to sync items.
	SyncParams struct {
		Params
		ComputeIntegrity bool          `json:"compute_integrity"`
		Limit            int           `json:"limit"`
		SyncToken        string        `json:"sync_token"`
		CursorToken      string        `json:"cursor_token"`
		ContentType      string        `json:"content_type"` // optional, only return items of these type if present
		Items            []*model.Item `json:"items"`
	}

	// A SyncService is a service used for syncing items.
	SyncService interface {
		// Execute performs the synchronisation.
		Execute() error
	}

	syncServiceBase struct {
		db     database.Client
		User   *model.User `json:"-"`
		Params SyncParams  `json:"-"`
	}

	errorItem struct {
		Message string `json:"message"`
		Tag     string `json:"tag"`
	}
)

// NewSync instantiates a new Sync service.
func NewSync(db database.Client, user *model.User, params SyncParams) (s SyncService) {
	switch params.APIVersion {
	case "20200115":
		fallthrough
	case "20190520":
		s = &syncService20190520{
			Base: &syncServiceBase{
				db:     db,
				User:   user,
				Params: params,
			},
		}
	case "20161215":
		fallthrough
	default:
		s = &syncService20161215{
			Base: &syncServiceBase{
				db:     db,
				User:   user,
				Params: params,
			},
		}
	}
	return s
}

// Get
func (s *syncServiceBase) get() ([]*model.Item, bool, error) {
	if s.Params.SyncToken == "" {
		// If it's the first sync request, front-load all exisitng items keys
		// so that the client can decrypt incoming items without having to wait.
		s.Params.Limit = 0
	}

	var (
		updated   time.Time
		strict    bool
		noDeleted bool
	)

	// if both are present, cursor_token takes precedence as that would eventually return all results
	// the distinction between getting results for a cursor and a sync token is that cursor results use a
	// >= comparison, while a sync token uses a > comparison. The reason for this is that cursor tokens are
	// typically used for initial syncs or imports, where a bunch of notes could have the exact same updated_at
	// by using >=, we don't miss those results on a subsequent call with a cursor token.
	switch {
	case s.Params.CursorToken != "":
		updated = libsf.TimeFromToken(s.Params.CursorToken)
	case s.Params.SyncToken != "":
		updated = libsf.TimeFromToken(s.Params.SyncToken)
		strict = true
	default:
		// if no cursor token and no sync token, this is an initial sync. No need to return deleted items.
		noDeleted = true
	}

	return s.db.FindItemsByParams(
		s.User.ID, s.Params.ContentType,
		updated, strict,
		noDeleted, s.Params.Limit)
}

// Compute data signature for integrity check
//
// https://github.com/standardfile/sfjs/blob/499fd0bc7ebddfc72f8b1dc3c9cbf134e92016d3/lib/app/lib/modelManager.js#L664-L677
func (s *syncServiceBase) computeDataSignature() (string, error) {
	items, err := s.db.FindItemsForIntegrityCheck(s.User.ID)
	if err != nil {
		return "", err
	}

	timestamps := []string{}
	for _, item := range items {
		// Unix timestamp in milliseconds (like MRI's `Time.now.to_datetime.strftime('%Q')`)
		timestamps = append(timestamps, fmt.Sprintf("%d", item.UpdatedAt.UnixMilli()))
	}

	sort.SliceStable(timestamps, func(i, j int) bool {
		return timestamps[j] < timestamps[i]
	})

	b := []byte(strings.Join(timestamps, ","))
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

// PrepareDelete
func (s *syncServiceBase) prepareDelete(item *model.Item) {
	item.Content = ""
	item.EncryptedItemKey = ""
}
