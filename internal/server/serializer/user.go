package serializer

import (
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

// User serializes the render of a user.
func User(m *model.User) map[string]any {
	r := map[string]any{
		"uuid":       m.ID,
		"created_at": m.CreatedAt.UTC(),
		"updated_at": m.UpdatedAt.UTC(),
		"email":      m.Email,
		"version":    m.Version,
		"pw_cost":    m.PasswordCost,
	}

	switch m.Version {
	case libsf.ProtocolVersion2:
		r["pw_salt"] = m.PasswordSalt
		r["pw_auth"] = m.PasswordAuth
	case libsf.ProtocolVersion3:
		fallthrough
	case libsf.ProtocolVersion4:
		r["pw_nonce"] = m.PasswordNonce
	}

	return r
}
