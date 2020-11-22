package libsf

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

// A KeyChain contains all the keys used for encryption and authentication.
type KeyChain struct {
	Version   string
	Password  string // Servers' password
	MasterKey string
	AuthKey   string            // Before protocol 004
	ItemsKey  map[string]string // Since protocol 004
}

func contentKeyChain(version string, k string) *KeyChain {
	switch version {
	case ProtocolVersion2:
		fallthrough
	case ProtocolVersion3:
		// Split item key in encryption key and auth key
		return &KeyChain{Version: version, MasterKey: k[:len(k)/2], AuthKey: k[len(k)/2:]}
	case ProtocolVersion4:
		return &KeyChain{Version: version, MasterKey: k}
	}

	return &KeyChain{}
}

// GenerateItemEncryptionKey generates a key used to encrypt item's content.
// ProtocolVersion3 is a 512 length bytes key that will be split in half, each being 256 bits.
// ProtocolVersion4 is a 32 length bytes key that be used as is.
func (k *KeyChain) GenerateItemEncryptionKey() (string, error) {
	switch k.Version {
	case ProtocolVersion2:
		fallthrough
	case ProtocolVersion3:
		passphrase, err := GenerateRandomBytes(512)
		if err != nil {
			return "", errors.Wrap(err, "vaut3: passphrase")
		}

		salt, err := GenerateRandomBytes(512)
		if err != nil {
			return "", errors.Wrap(err, "vaut3: salt")
		}

		ik := pbkdf2.Key(passphrase, salt, 1, 2*32, sha512.New)
		return hex.EncodeToString(ik), nil
		//
		//
	case ProtocolVersion4:
		ik, err := GenerateRandomBytes(32)
		if err != nil {
			return "", errors.Wrap(err, "vaut4: key")
		}
		return hex.EncodeToString(ik), nil
	}

	return "", errors.Errorf("Unsupproted version: %s", k.Version)
}
