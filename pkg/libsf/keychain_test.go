package libsf_test

import (
	"testing"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestKeychain_GenerateItemEncryptionKey(t *testing.T) {
	keychain := &libsf.KeyChain{}

	keychain.Version = libsf.ProtocolVersion2
	ik, err := keychain.GenerateItemEncryptionKey()
	assert.NoError(t, err)
	assert.Equal(t, 128, len(ik))

	keychain.Version = libsf.ProtocolVersion3
	ik, err = keychain.GenerateItemEncryptionKey()
	assert.NoError(t, err)
	assert.Equal(t, 128, len(ik))

	keychain.Version = libsf.ProtocolVersion4
	ik, err = keychain.GenerateItemEncryptionKey()
	assert.NoError(t, err)
	assert.Equal(t, 64, len(ik))
}
