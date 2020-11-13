package libsf

import "strconv"

const (
	// APIVersion20161215 allows to use the API version 20161215.
	APIVersion20161215 = "20161215"
	// APIVersion20190520 allows to use the API version 20190520.
	APIVersion20190520 = "20190520"
	// APIVersion20200115 allows to use the API version 20200115.
	APIVersion20200115 = "20200115"

	// APIVersion is the version used by default client.
	APIVersion = APIVersion20190520
)

const (
	// ProtocolVersion1 allows to use the SF protocol 001.
	ProtocolVersion1 = "001"
	// ProtocolVersion2 allows to use the SF protocol 002.
	ProtocolVersion2 = "002"
	// ProtocolVersion3 allows to use the SF protocol 003.
	ProtocolVersion3 = "003"
	// ProtocolVersion4 allows to use the SF protocol 004.
	ProtocolVersion4 = "004"
)

// VersionGreaterOrEqual returns true if current is not empty and greater or equal to version.
func VersionGreaterOrEqual(version, current string) bool {
	if current == "" {
		return false
	}

	v, err := strconv.Atoi(version)
	if err != nil {
		return false
	}

	c, err := strconv.Atoi(current)
	if err != nil {
		return false
	}

	return c >= v
}

// VersionLesser returns true if current is empty or lesser to version.
func VersionLesser(version, current string) bool {
	if current == "" {
		return true
	}

	v, err := strconv.Atoi(version)
	if err != nil {
		return false
	}

	c, err := strconv.Atoi(current)
	if err != nil {
		return false
	}

	return c < v
}
