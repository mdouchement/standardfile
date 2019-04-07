package service

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

const minConflictInterval = 20

type (
	// A SyncParams is used when a client want to sync items.
	SyncParams struct {
		ComputeIntegrity bool          `json:"compute_integrity"`
		Limit            int           `json:"limit"`
		SyncToken        string        `json:"sync_token"`
		CursorToken      string        `json:"cursor_token"`
		ContentType      string        `json:"content_type"` // optional, only return items of these type if present
		Items            []*model.Item `json:"items"`
	}

	// A SyncService is a service used for syncing items.
	SyncService struct {
		db     database.Client
		User   *model.User `json:"-"`
		Params SyncParams  `json:"-"`
		// Populated during `Execute()`
		Retrieved     []*model.Item  `json:"retrieved_items"`
		Saved         []*model.Item  `json:"saved_items"`
		Unsaved       []*UnsavedItem `json:"unsaved"`
		SyncToken     string         `json:"sync_token"`
		CursorToken   string         `json:"cursor_token"`
		IntegrityHash string         `json:"integrity_hash,omitempty"`
	}

	// An UnsavedItem is an object containing an item that can't be saved.
	UnsavedItem struct {
		Item  *model.Item `json:"item"`
		Error errorItem   `json:"error"`
	}

	errorItem struct {
		Message string `json:"message"`
		Tag     string `json:"tag"`
	}
)

// NewSync instantiates a new Sync service.
func NewSync(db database.Client, user *model.User, params SyncParams) *SyncService {
	return &SyncService{
		db:     db,
		User:   user,
		Params: params,
	}
}

// Execute performs the synchronisation.
func (s *SyncService) Execute() error {
	retrievedItems, overLimit, err := s.get()
	if err != nil {
		return err
	}
	s.Retrieved = retrievedItems

	s.Saved, s.Unsaved = s.save()

	retrievedToDelete := s.checkForConflicts()
	// Remove potential conflicted items => cf. checkForConflicts()
	var n int
	for _, item := range s.Retrieved {
		if retrievedToDelete[item.ID] {
			continue
		}

		s.Retrieved[n] = item
		n++
	}
	s.Retrieved = s.Retrieved[:n]

	// In reference implementation, there is post_to_extensions but not implemented here.
	// See README.md

	if s.Params.ComputeIntegrity {
		s.IntegrityHash, err = s.computeDataSignature()
		if err != nil {
			return err
		}
	}

	//
	// Prepare returned value
	//

	// CursorToken
	if overLimit {
		s.CursorToken = libsf.TokenFromTime(*retrievedItems[s.Params.Limit-1].UpdatedAt)
	}

	// SyncToken
	var lastUpdated time.Time
	for _, item := range s.Saved {
		if item.UpdatedAt.After(lastUpdated) {
			lastUpdated = *item.UpdatedAt
		}
	}
	if lastUpdated.IsZero() { // occurred when `len(savedItems) == 0'
		lastUpdated = time.Now()
	}

	// add 1 microsecond to avoid returning same object in subsequent sync.
	s.SyncToken = libsf.TokenFromTime(lastUpdated.Add(1 * time.Nanosecond))

	return nil
}

//
// Get
//
func (s *SyncService) get() ([]*model.Item, bool, error) {
	if s.Params.Limit == 0 {
		s.Params.Limit = 100000
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

//
// Save
//
func (s *SyncService) save() (saved []*model.Item, unsaved []*UnsavedItem) {
	saved = make([]*model.Item, 0)
	unsaved = make([]*UnsavedItem, 0)

	if len(s.Params.Items) == 0 {
		return
	}

	for _, item := range s.Params.Items {
		item.UserID = s.User.ID

		if item.Deleted {
			s.prepareDelete(item)
		}

		err := s.db.Save(item)
		if err != nil {
			// TODO return an Internal Server Error?
			unsaved = append(unsaved, &UnsavedItem{
				Item: item,
				Error: errorItem{
					Message: err.Error(),
					// There is no need of the tag. `Save` will insert or update.
					// https://github.com/standardfile/rails-engine/blob/cc0d40856800ab1fa9fd1aa20a03e98f8d351a0b/lib/standard_file/sync_manager.rb#L118-L123
					// Tag: "uuid_conflict",
				},
			})
		}

		saved = append(saved, item)
	}

	return
}

//
// Check conflicts
//
func (s *SyncService) checkForConflicts() map[string]bool {
	// Saved is the smallest slice.
	saved := make(map[string]*model.Item)
	for _, item := range s.Saved {
		saved[item.ID] = item
	}

	retrieved := make(map[string]*model.Item)
	for _, item := range s.Retrieved {
		if _, ok := saved[item.ID]; ok {
			// Keep only items within the intersection.
			// There are the conflicted items to iterate on and compare to the saved one.
			retrieved[item.ID] = item
		}
	}

	tobedeleted := map[string]bool{}
	// Saved items take precedence, retrieved items are duplicated with a new uuid.
	for id, conflicted := range retrieved {
		diff := saved[id].UpdatedAt.Sub(*conflicted.UpdatedAt).Seconds()
		diff = math.Abs(diff)

		// If changes are greater than minConflictInterval seconds apart,
		// create conflicted copy, otherwise discard conflicted.
		if diff > minConflictInterval {
			s.Unsaved = append(s.Unsaved, &UnsavedItem{
				Item: conflicted,
				Error: errorItem{
					Tag: "sync_conflict",
				},
			})
		}

		// We remove the item from retrieved items whether or not it satisfies the minConflictInterval.
		// This is because the 'saved' value takes precedence, since that's the current value in the database.
		// So by removing it from retrieved, we are forcing the client to ignore this change.
		tobedeleted[id] = true
	}

	return tobedeleted
}

//
// Compute data signature for integrity check
//
func (s *SyncService) computeDataSignature() (string, error) {
	items, err := s.db.FindItemsForIntegrityCheck(s.User.ID)
	if err != nil {
		return "", err
	}

	timestamps := []string{}
	for _, item := range items {
		// Unix timestamp in milliseconds (like MRI's `Time.now.to_datetime.strftime('%Q')`)
		timestamps = append(timestamps, fmt.Sprintf("%d", item.UpdatedAt.UnixNano()/1000000))
	}

	sort.SliceStable(timestamps, func(i, j int) bool {
		return timestamps[j] < timestamps[i]
	})

	b := []byte(strings.Join(timestamps, ","))
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

//
// PrepareDelete
//
func (s *SyncService) prepareDelete(item *model.Item) {
	item.Content = ""
	item.EncryptedItemKey = ""
}
