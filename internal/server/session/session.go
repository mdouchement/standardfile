package session

import (
	"strconv"

	"github.com/mdouchement/standardfile/internal/model"
)

// SessionProtocolVersion is the account version starting the support of sessions.
const SessionProtocolVersion = "004"

// UserSupportsJWT returns true if the user supports the JWT authentication model.
func UserSupportsJWT(user *model.User) bool {
	spv, err := strconv.Atoi(SessionProtocolVersion)
	if err != nil {
		return false
	}

	uv, err := strconv.Atoi(user.Version)
	if err != nil {
		return false
	}

	return uv < spv
}

// UserSupportsSessions returns true if the user supports the sessions authentication model.
func UserSupportsSessions(user *model.User) bool {
	spv, err := strconv.Atoi(SessionProtocolVersion)
	if err != nil {
		return false
	}

	uv, err := strconv.Atoi(user.Version)
	if err != nil {
		return false
	}

	return uv >= spv
}
