package service

import "github.com/mdouchement/standardfile/internal/model"

// M is an arbitrary map.
type M map[string]any

// Params are the basic fields used in requests.
type Params struct {
	APIVersion string `json:"api"` // Since 20190520
	UserAgent  string
	Session    *model.Session
}
