package service

// Params are the basic fields used in requests.
type Params struct {
	APIVersion string `json:"api"` // Since 20190520
	UserAgent  string
}
