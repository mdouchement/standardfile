package libsf

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fastjson"
)

const noteTimeLayout = "2006-01-02T15:04:05.000Z"

// A Note is plaintext Item content.
type Note struct {
	Title        string          `json:"title"`
	Text         string          `json:"text"`
	PreviewPlain string          `json:"preview_plain"`
	PreviewHTML  string          `json:"preview_html"`
	References   json.RawMessage `json:"references"` // unstructured data
	AppData      json.RawMessage `json:"appData"`    // unstructured data

	appdata *fastjson.Value
}

// ParseRaw parses unstructured raw fields.
// Needed before using other methods on Note object.
func (n *Note) ParseRaw() error {
	v, err := fastjson.Parse(string(n.AppData))
	if err != nil {
		return errors.Wrap(err, "could not parse raw data")
	}

	n.appdata = v
	return nil
}

// SaveRaw persists the unstructured fields to raw data.
func (n *Note) SaveRaw() {
	n.AppData = json.RawMessage(n.appdata.String())
}

// UpdatedAt returns the last update time of the note.
// If not found or error, "zero" time is returned.
func (n *Note) UpdatedAt() time.Time {
	if n.appdata.Exists("org.standardnotes.sn", "client_updated_at") {
		s := string(n.appdata.GetStringBytes("org.standardnotes.sn", "client_updated_at"))

		t, err := time.Parse(noteTimeLayout, s)
		if err != nil {
			return time.Time{}
		}

		return t
	}

	return time.Time{}
}

// SetUpdatedAtNow sets current time as last update time.
func (n *Note) SetUpdatedAtNow() {
	s := time.Now().Format(noteTimeLayout)

	n.appdata.
		Get("org.standardnotes.sn").
		Set("client_updated_at", new(fastjson.Arena).NewString(s))
}

// GetSortingField returns the field on which all notes are sorted.
// Only for `SN|UserPreferences` items, it returns an empty string if nothing found.
func (n *Note) GetSortingField() string {
	if n.appdata.Exists("org.standardnotes.sn", "sortBy") {
		return string(
			n.appdata.Get("org.standardnotes.sn", "sortBy").GetStringBytes(),
		)
	}
	return ""
}
