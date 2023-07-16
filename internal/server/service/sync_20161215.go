package service

import (
	"math"
	"time"

	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

const minConflictInterval20161215 = 20 // in second

type (
	// A syncService20161215 is a service used for syncing items.
	syncService20161215 struct {
		Base *syncServiceBase `json:"-"`
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
)

// Execute performs the synchronisation.
func (s *syncService20161215) Execute() error {
	retrievedItems, overLimit, err := s.Base.get()
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

	if s.Base.Params.ComputeIntegrity {
		s.IntegrityHash, err = s.Base.computeDataSignature()
		if err != nil {
			return err
		}
	}

	//
	// Prepare returned value
	//

	// CursorToken
	if overLimit {
		s.CursorToken = libsf.TokenFromTime(*retrievedItems[s.Base.Params.Limit-1].UpdatedAt)
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
	s.SyncToken = libsf.TokenFromTime(lastUpdated.Add(1 * time.Microsecond))

	return nil
}

// Save
func (s *syncService20161215) save() (saved []*model.Item, unsaved []*UnsavedItem) {
	saved = make([]*model.Item, 0)
	unsaved = make([]*UnsavedItem, 0)

	if len(s.Base.Params.Items) == 0 {
		return
	}

	for _, item := range s.Base.Params.Items {
		item.UserID = s.Base.User.ID

		if item.Deleted {
			s.Base.prepareDelete(item)
		}

		err := s.Base.db.Save(item) // aka item.update(..)
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
			continue
		}

		saved = append(saved, item)
	}

	return
}

// Check conflicts
func (s *syncService20161215) checkForConflicts() map[string]bool {
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

		// If changes are greater than minConflictInterval20161215 seconds apart,
		// create conflicted copy, otherwise discard conflicted.
		if diff > minConflictInterval20161215 {
			s.Unsaved = append(s.Unsaved, &UnsavedItem{
				Item: conflicted,
				Error: errorItem{
					Tag: "sync_conflict",
				},
			})
		}

		// We remove the item from retrieved items whether or not it satisfies the minConflictInterval20161215.
		// This is because the 'saved' value takes precedence, since that's the current value in the database.
		// So by removing it from retrieved, we are forcing the client to ignore this change.
		tobedeleted[id] = true
	}

	return tobedeleted
}
