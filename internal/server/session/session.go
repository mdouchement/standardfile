package session

import (
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

// SessionProtocolVersion is the account version starting the support of sessions.
const SessionProtocolVersion = libsf.ProtocolVersion4

// UserSupportsJWT returns true if the user supports the JWT authentication model.
func UserSupportsJWT(user *model.User) bool {
	return libsf.VersionLesser(SessionProtocolVersion, user.Version)
}

// UserSupportsSessions returns true if the user supports the sessions authentication model.
func UserSupportsSessions(user *model.User) bool {
	return libsf.VersionGreaterOrEqual(SessionProtocolVersion, user.Version)
}
