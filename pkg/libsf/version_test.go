package libsf_test

import (
	"testing"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestVersionGreaterOrEqual(t *testing.T) {
	assert.False(t, libsf.VersionGreaterOrEqual("bad", "1"))
	assert.False(t, libsf.VersionGreaterOrEqual("1", "bad"))

	assert.False(t, libsf.VersionGreaterOrEqual(libsf.ProtocolVersion3, libsf.ProtocolVersion2))
	assert.True(t, libsf.VersionGreaterOrEqual(libsf.ProtocolVersion3, libsf.ProtocolVersion3))
	assert.True(t, libsf.VersionGreaterOrEqual(libsf.ProtocolVersion3, libsf.ProtocolVersion4))

	assert.False(t, libsf.VersionGreaterOrEqual(libsf.APIVersion20190520, libsf.APIVersion20161215))
	assert.True(t, libsf.VersionGreaterOrEqual(libsf.APIVersion20190520, libsf.APIVersion20190520))
	assert.True(t, libsf.VersionGreaterOrEqual(libsf.APIVersion20190520, libsf.APIVersion20200115))
}

func TestVersionLesser(t *testing.T) {
	assert.False(t, libsf.VersionLesser("bad", "1"))
	assert.False(t, libsf.VersionLesser("1", "bad"))

	assert.True(t, libsf.VersionLesser(libsf.ProtocolVersion3, libsf.ProtocolVersion2))
	assert.False(t, libsf.VersionLesser(libsf.ProtocolVersion3, libsf.ProtocolVersion3))
	assert.False(t, libsf.VersionLesser(libsf.ProtocolVersion3, libsf.ProtocolVersion4))

	assert.True(t, libsf.VersionLesser(libsf.APIVersion20190520, libsf.APIVersion20161215))
	assert.False(t, libsf.VersionLesser(libsf.APIVersion20190520, libsf.APIVersion20190520))
	assert.False(t, libsf.VersionLesser(libsf.APIVersion20190520, libsf.APIVersion20200115))
}
