package serializer

import "github.com/mdouchement/standardfile/internal/model"

// Session serializes the render of a session.
func Session(m *model.Session) map[string]interface{} {
	r := map[string]interface{}{
		"uuid":       m.ID,
		"created_at": m.CreatedAt,
		"updated_at": m.UpdatedAt,
		"email":      m.UserAgent,
		"version":    m.APIVersion,
		"current":    m.Current,
	}
	return r
}

// Sessions serializes the render of sessions.
func Sessions(m []*model.Session) []map[string]interface{} {
	sessions := make([]map[string]interface{}, len(m))
	for i, s := range m {
		sessions[i] = Session(s)
	}
	return sessions
}
