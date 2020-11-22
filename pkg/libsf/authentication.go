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
		// Identifier returns the identifier (email) used for authentication.
		Identifier() string
		// Version returns the encryption scheme version.
		Version() string
		// IntegrityCheck checks if the Auth params are valid.
		IntegrityCheck() error
		// SymmetricKeyPair returns a KeyChain for the given uip (plaintext password of the user).
		// https://github.com/standardfile/standardfile.github.io/blob/master/doc/spec.md#client-instructions
		SymmetricKeyPair(uip string) *KeyChain
	}

	auth struct {
		FieldVersion     string `json:"version"`
		FieldIdentifier  string `json:"identifier"`
		FieldCost        int    `json:"pw_cost,omitempty"` // Before protocol 004
		FieldNonce       string `json:"pw_nonce"`
		FieldOrigination string `json:"origination,omitempty"` // Since protocol 004
	}
)

func (a *auth) Email() string {
	return a.FieldIdentifier
}

func (a *auth) Identifier() string {
	return a.FieldIdentifier
}

func (a *auth) Version() string {
	return a.FieldVersion
}

func (a *auth) IntegrityCheck() error {
	switch a.FieldVersion {
	case ProtocolVersion4:
		// nothing
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

func (a *auth) SymmetricKeyPair(uip string) *KeyChain {
	switch a.FieldVersion {
	case ProtocolVersion4:
		return a.SymmetricKeyPair4(uip)
	default:
		return a.SymmetricKeyPair3(uip)
	}
}

func (a *auth) SymmetricKeyPair3(uip string) *KeyChain {
	token := fmt.Sprintf("%s:SF:%s:%d:%s", a.FieldIdentifier, a.FieldVersion, a.FieldCost, a.FieldNonce)
	salt := fmt.Sprintf("%x", sha256.Sum256([]byte(token))) // Hexadecimal sum

	// We need 3 keys of 32 length each.
	k := pbkdf2.Key([]byte(uip), []byte(salt), a.FieldCost, 3*32, sha512.New)
	key := hex.EncodeToString(k)
	s := len(key) / 3

	return &KeyChain{
		Version:   a.FieldVersion,
		Password:  key[:s],
		MasterKey: key[s : s*2],
		AuthKey:   key[s*2:],
	}
}

func (a *auth) SymmetricKeyPair4(uip string) *KeyChain {
	payload := fmt.Sprintf("%s:%s", a.FieldIdentifier, a.FieldNonce)
	hash := sha256.Sum256([]byte(payload))
	// Taking the first 16 bytes of the hash is the same
	// as taking the 32 first characters of the hexa salt as described in specifications.
	salt := hash[:16]

	key, _ := kdf4([]byte(uip), salt)
	return &KeyChain{
		Version:   a.FieldVersion,
		MasterKey: hex.EncodeToString(key[:32]),
		Password:  hex.EncodeToString(key[32:]),
		ItemsKey:  map[string]string{},
	}
}
