package model

import (
	"time"
)

type (
	// A Model defines an object that can be stored in database.
	Model interface {
		// GetID returns the model's ID.
		GetID() string
		// SetID defines the model's ID.
		SetID(string)
		// GetCreatedAt returns the model's creation date.
		GetCreatedAt() *time.Time
		// SetCreatedAt defines the model's creation date.
		SetCreatedAt(time.Time)
		// GetUpdatedAt returns the model's last update date.
		GetUpdatedAt() *time.Time
		// SetUpdatedAt defines the model's last update date.
		SetUpdatedAt(time.Time)
	}

	// A Base contains the default model fields.
	Base struct {
		ID        string     `json:"uuid"       msgpack:"id"         storm:"id"`
		CreatedAt *time.Time `json:"created_at" msgpack:"created_at" storm:"index"`
		UpdatedAt *time.Time `json:"updated_at" msgpack:"updated_at" storm:"index"`
	}
)

// GetID returns the model's ID.
func (m *Base) GetID() string {
	return m.ID
}

// SetID defines the model's ID.
func (m *Base) SetID(id string) {
	m.ID = id
}

// GetCreatedAt returns the model's creation date.
func (m *Base) GetCreatedAt() *time.Time {
	return m.CreatedAt
}

// SetCreatedAt defines the model's creation date.
func (m *Base) SetCreatedAt(t time.Time) {
	m.CreatedAt = &t
}

// GetUpdatedAt returns the model's last update date.
func (m *Base) GetUpdatedAt() *time.Time {
	return m.UpdatedAt
}

// SetUpdatedAt defines the model's last update date.
func (m *Base) SetUpdatedAt(t time.Time) {
	m.UpdatedAt = &t
}
