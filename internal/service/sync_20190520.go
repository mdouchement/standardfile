package service

import (
	"math"
	"time"

	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/sirupsen/logrus"
)

// Ignore differences that are at most this many seconds apart
// Anything over this threshold will be conflicted.
const minConflictInterval20190520 = 1.0 // in second

type (
	// A syncService20190520 is a service used for syncing items.
	syncService20190520 struct {
		Base *syncServiceBase
		// Populated during `Execute()`
		Retrieved     []*model.Item   `json:"retrieved_items"`
		Saved         []*model.Item   `json:"saved_items"`
		Conflicts     []*ConflictItem `json:"conflicts"`
		SyncToken     string          `json:"sync_token"`
		CursorToken   string          `json:"cursor_token"`
		IntegrityHash string          `json:"integrity_hash,omitempty"`
	}

	// An ConflictItem is an object containing an item that can't be saved caused by conflicts.
	ConflictItem struct {
		UnsavedItem *model.Item `json:"unsaved_item,omitempty"`
		ServerItem  *model.Item `json:"server_item,omitempty"`
		Type        string      `json:"type"`
	}
)

// Execute performs the synchronisation.
func (s *syncService20190520) Execute() error {
	retrievedItems, overLimit, err := s.Base.get()
	if err != nil {
		return err
	}
	s.Retrieved = retrievedItems

	var retrievedToDelete map[string]bool
	s.Saved, s.Conflicts, retrievedToDelete = s.save()

	// Remove potential conflicted items
	var n int
	for _, item := range s.Retrieved {
		if retrievedToDelete[item.GetID()] {
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

//
// Save
//
func (s *syncService20190520) save() (saved []*model.Item, conflicts []*ConflictItem, tobedeleted map[string]bool) {
	saved = make([]*model.Item, 0)
	conflicts = make([]*ConflictItem, 0)
	tobedeleted = map[string]bool{}

	if len(s.Base.Params.Items) == 0 {
		return
	}

	for _, incomingItem := range s.Base.Params.Items {
		incomingItem.UserID = s.Base.User.ID

		serverItem, err := s.Base.db.FindItemByUserID(incomingItem.GetID(), s.Base.User.ID)
		newRecord := s.Base.db.IsNotFound(err)
		if err != nil && !newRecord {
			// TODO: return an Internal Server Error?
			logrus.WithError(err).Error("could not find item")
			conflicts = append(conflicts, &ConflictItem{
				UnsavedItem: incomingItem,
				Type:        "internal_error", // FIXME: do not exists in reference implementation.
			})
			continue
		}

		if !newRecord {
			// We want to check if this updated_at value is equal to the item's current updated_at value.
			// If they differ, it means the client is attempting to save an item which hasn't been updated.
			// In this case, if the incoming_item.updated_at < server_item.updated_at, always conflict.
			// We don't want old items overriding newer ones.
			// incoming_item.updated_at > server_item.updated_at would seem to be impossible, as only servers are responsible for setting updated_at.
			// But assuming a rogue client has gotten away with it,
			// we should also conflict in this case if the difference between the dates is greater than MIN_CONFLICT_INTERVAL seconds.

			saveIncoming := true
			// SFJS did not send updated_at prior to 0.3.59 but applied by the database layer so the value is OK.
			difference := incomingItem.UpdatedAt.Sub(*serverItem.UpdatedAt).Seconds()

			switch {
			case difference < 0:
				// incoming is less than server item. This implies stale data. Don't save if greater than interval
				fallthrough
			case difference > 0:
				// incoming is greater than server item. Should never be the case. If so though, don't save.
				saveIncoming = math.Abs(difference) < minConflictInterval20190520
			default:
				// incoming is equal to server item (which is desired, healthy behavior), continue with saving.
				saveIncoming = true
			}

			if !saveIncoming {
				// Dont save incoming and send it back. At this point the server item is likely to be included
				// in retrievedItems in a subsequent sync, so when that value comes into the client.
				conflicts = append(conflicts, &ConflictItem{
					ServerItem: serverItem,
					Type:       "sync_conflict",
				})
				tobedeleted[serverItem.GetID()] = true
				continue
			}
		}

		if incomingItem.Deleted {
			s.Base.prepareDelete(incomingItem)
		}

		err = s.Base.db.Save(incomingItem) // aka item.update(..)
		if err != nil {
			// TODO: return an Internal Server Error?
			// Type is pretty useless because `Save` will insert or update.
			logrus.WithError(err).Error("could not save item")
			conflicts = append(conflicts, &ConflictItem{
				UnsavedItem: incomingItem,
				Type:        "uuid_conflict",
			})
			continue
		}

		saved = append(saved, incomingItem)
	}

	return
}
