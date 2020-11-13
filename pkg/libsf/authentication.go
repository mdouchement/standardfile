package libsf

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrUnsupportedVersion is raised when user params version is lesser than `002`.
	ErrUnsupportedVersion = errors.New("libsf: unsupported version")
	// ErrLowPasswordCost occurred when cost of password is too low for the used KDF.
	ErrLowPasswordCost = errors.New("libsf: low password cost")
)

type (
	// An Auth holds all the params needed to create the credentials and cipher keys.
	Auth interface {
		// Email returns the email used for authentication.
		Email() string
		// Version returns the encryption scheme version.
		Version() string
		// IntegrityCheck checks if the Auth params are valid.
		IntegrityCheck() error
		// SymmetricKeyPair returns the password, master_key and auth_key for the given uip (plaintext password of the user).
		// https://github.com/standardfile/standardfile.github.io/blob/master/doc/spec.md#client-instructions
		SymmetricKeyPair(uip string) (pw, mk, ak string)
	}

	auth struct {
		FieldEmail   string `json:"identifier"`
		FieldVersion string `json:"version"`
		FieldCost    int    `json:"pw_cost"`
		FieldNonce   string `json:"pw_nonce"`
	}
)

func (a *auth) Email() string {
	return a.FieldEmail
}

func (a *auth) Version() string {
	return a.FieldVersion
}

func (a *auth) IntegrityCheck() error {
	switch a.FieldVersion {
	case ProtocolVersion3:
		if a.FieldCost < 110000 {
			return ErrLowPasswordCost
		}
	case ProtocolVersion2:
		if a.FieldCost < 3000 {
			return ErrLowPasswordCost
		}
	case ProtocolVersion1:
		fallthrough
	default:
		return ErrUnsupportedVersion
	}

	return nil
}

func (a *auth) SymmetricKeyPair(uip string) (pw, mk, ak string) {
	token := fmt.Sprintf("%s:SF:%s:%d:%s", a.FieldEmail, a.FieldVersion, a.FieldCost, a.FieldNonce)
	salt := fmt.Sprintf("%x", sha256.Sum256([]byte(token))) // Hexadecimal sum

	// We need 3 keys of 32 length each.
	k := pbkdf2.Key([]byte(uip), []byte(salt), a.FieldCost, 3*32, sha512.New)
	key := hex.EncodeToString(k)
	s := len(key) / 3

	return key[:s], key[s : s*2], key[s*2:]
}
