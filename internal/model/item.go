package model

// A Item represents a database record and the rendered API response.
type Item struct {
	Base `msgpack:",inline" storm:"inline"`

	UserID           string `json:"user_uuid"    msgpack:"user_id"      storm:"index"`
	Content          string `json:"content"      msgpack:"content"`
	ContentType      string `json:"content_type" msgpack:"content_type" storm:"index"`
	EncryptedItemKey string `json:"enc_item_key" msgpack:"enc_item_key"`
	Deleted          bool   `json:"deleted"      msgpack:"deleted"      storm:"index"`
}
