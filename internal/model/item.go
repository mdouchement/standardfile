package model

import "time"

// A Item represents a database record and the rendered API response.
type Item struct {
	Base `msgpack:",inline" storm:"inline"`

	EditedAt         *time.Time `json:"edited_at" msgpack:"edited_at" storm:"index"`
	UserID           string     `json:"user_uuid"    msgpack:"user_id"                 storm:"index"`
	ItemsKeyID       string     `json:"items_key_id" msgpack:"items_key_id,omitempty"`
	Content          string     `json:"content"      msgpack:"content"`
	ContentType      string     `json:"content_type" msgpack:"content_type"            storm:"index"`
	EncryptedItemKey string     `json:"enc_item_key" msgpack:"enc_item_key"`
	Deleted          bool       `json:"deleted"      msgpack:"deleted"                 storm:"index"`
}
