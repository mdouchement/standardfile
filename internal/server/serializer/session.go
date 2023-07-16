package serializer

import "github.com/mdouchement/standardfile/internal/model"

// Session serializes the render of a session.
func Session(m *model.Session) map[string]any {
	r := map[string]any{
		"uuid":        m.ID,
		"created_at":  m.CreatedAt.UTC(),
		"updated_at":  m.UpdatedAt.UTC(),
		"api_version": m.APIVersion,
		"user_agent":  m.UserAgent, // TODO: rename field to device_info?
		"current":     m.Current,
	}
	return r
}

// Sessions serializes the render of sessions.
func Sessions(m []*model.Session) []map[string]any {
	sessions := make([]map[string]any, len(m))
	for i, s := range m {
		sessions[i] = Session(s)
	}
	return sessions
}
