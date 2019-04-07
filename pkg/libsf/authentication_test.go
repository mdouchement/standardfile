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
			version: "003",
			cost:    110000,
			err:     nil,
		},
		{
			version: "003",
			cost:    109999,
			err:     libsf.ErrLowPasswordCost,
		},
		{
			version: "002",
			cost:    3000,
			err:     nil,
		},
		{
			version: "002",
			cost:    2999,
			err:     libsf.ErrLowPasswordCost,
		},
		{
			version: "001",
			cost:    3000,
			err:     libsf.ErrUnsupportedVersion,
		},
		{
			version: "001",
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
	auth := libsf.NewAuth("george.abitbol@nowhere.lan", "003", "nonce", 420000)
	pw, mk, ak := auth.SymmetricKeyPair("password42")

	assert.Equal(t, "91fe137892ea5016105162767c66088474f47eee039187d695bc129cc01afc6e", pw)
	assert.Equal(t, "b0edcb1b9bcdfe797a557c47a0045d72c2ad06bbc3e47f98b3676a8284a895fd", mk)
	assert.Equal(t, "3ef83ac304168b6950ca059365a3b8d00d251b8d67ef3965210c20207de388dd", ak)
}
