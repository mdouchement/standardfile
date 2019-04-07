package libsf

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

// GenerateItemEncryptionKey generates a key that will be split in half, each being 256 bits.
// So total length will need to be 512.
func GenerateItemEncryptionKey() (string, error) {
	passphrase, err := GenerateRandomBytes(512)
	if err != nil {
		return "", errors.Wrap(err, "passphrase")
	}

	salt, err := GenerateRandomBytes(512)
	if err != nil {
		return "", errors.Wrap(err, "salt")
	}

	ik := pbkdf2.Key(passphrase, salt, 1, 2*32, sha512.New)
	return hex.EncodeToString(ik), nil
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// err == nil only if len(b) == n
	if err != nil {
		return nil, err
	}

	return b, nil
}
