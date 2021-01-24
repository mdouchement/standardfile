package libsf_test

import (
	"testing"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestAuth_Email(t *testing.T) {
	email := "george.abitbol@nowhere.lan"
	auth := libsf.NewAuth(email, "003", "nonce", 42)

	assert.Equal(t, email, auth.Email())
}

func TestAuth_IntegrityCheck(t *testing.T) {
	data := []struct {
		version string
		cost    int
		err     error
	}{
		{
			version: libsf.ProtocolVersion3,
			cost:    110000,
			err:     nil,
		},
		{
			version: libsf.ProtocolVersion3,
			cost:    109999,
			err:     libsf.ErrLowPasswordCost,
		},
		{
			version: libsf.ProtocolVersion2,
			cost:    3000,
			err:     nil,
		},
		{
			version: libsf.ProtocolVersion2,
			cost:    2999,
			err:     libsf.ErrLowPasswordCost,
		},
		{
			version: libsf.ProtocolVersion1,
			cost:    3000,
			err:     libsf.ErrUnsupportedVersion,
		},
		{
			version: libsf.ProtocolVersion1,
			cost:    2999,
			err:     libsf.ErrUnsupportedVersion,
		},
		{
			version: "",
			cost:    3000,
			err:     libsf.ErrUnsupportedVersion,
		},
		{
			version: "",
			cost:    2999,
			err:     libsf.ErrUnsupportedVersion,
		},
	}

	for _, d := range data {
		auth := libsf.NewAuth("george.abitbol@nowhere.lan", d.version, "nonce", d.cost)

		assert.Equal(t, d.err, auth.IntegrityCheck())
	}
}

func TestAuth_SymmetricKeyPair(t *testing.T) {
	auth := libsf.NewAuth("george.abitbol@nowhere.lan", libsf.ProtocolVersion3, "nonce", 420000)
	keychain := auth.SymmetricKeyPair("password42")

	assert.Equal(t, libsf.ProtocolVersion3, keychain.Version)
	assert.Equal(t, "91fe137892ea5016105162767c66088474f47eee039187d695bc129cc01afc6e", keychain.Password)
	assert.Equal(t, "b0edcb1b9bcdfe797a557c47a0045d72c2ad06bbc3e47f98b3676a8284a895fd", keychain.MasterKey)
	assert.Equal(t, "3ef83ac304168b6950ca059365a3b8d00d251b8d67ef3965210c20207de388dd", keychain.AuthKey)

	//

	auth = libsf.NewAuth("george.abitbol@nowhere.lan", libsf.ProtocolVersion4, "nonce", 0)
	keychain = auth.SymmetricKeyPair("password42")

	assert.Equal(t, libsf.ProtocolVersion4, keychain.Version)
	assert.Equal(t, "d89dc5c8a7719daf1160b9f2f4d858fe3d51d960b44f1487c5e2002fbb0d7b2b", keychain.Password)
	assert.Equal(t, "e669ef0a61bc253a884201cc1aceabfff78f29d0515aeceb8f3fdb80e35c3f79", keychain.MasterKey)
	assert.Equal(t, "", keychain.AuthKey)

	// From SNJS tests
	auth = libsf.NewAuth("foo@bar.com", libsf.ProtocolVersion4, "baaec0131d677cf993381367eb082fe377cefe70118c1699cb9b38f0bc850e7b", 0)
	keychain = auth.SymmetricKeyPair("very_secure")

	assert.Equal(t, libsf.ProtocolVersion4, keychain.Version)
	assert.Equal(t, "83707dfc837b3fe52b317be367d3ed8e14e903b2902760884fd0246a77c2299d", keychain.Password)
	assert.Equal(t, "5d68e78b56d454e32e1f5dbf4c4e7cf25d74dc1efc942e7c9dfce572c1f3b943", keychain.MasterKey)
	assert.Equal(t, "", keychain.AuthKey)
}
