package libsf

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

type (
	vault4 struct {
		version        string
		nonce          string
		ciphertext     string
		rawauth        string
		auth           authenticatedData
		additionaldata string
	}

	authenticatedData struct {
		from      string
		Version   string `json:"v"`
		UserID    string `json:"u"`
		KeyParams auth   `json:"kp,omitempty"`
	}
)

////
///
//

func parse4(components []string) (vault, error) {
	v := &vault4{}

	if len(components) < 4 || len(components) > 5 {
		return v, errors.New("invalid secret format")
	}

	// https://github.com/standardnotes/app/blob/main/packages/snjs/specification.md#encryption---specifics
	v.version = components[0]
	v.nonce = components[1]
	v.ciphertext = components[2]
	v.rawauth = components[3]
	if len(components) > 4 {
		// This part is not defined in the specification but implemented in StandardNotes official implementation.
		// https://github.com/standardnotes/app/commit/b032eb9c9b4b98a1a256d3d03863866bb4136ec8#diff-cb607afd3ffe76488f6ba1f7885d16f853810e799c9f223a1dd7673d15928396
		v.additionaldata = components[4] // Default value is `e30=' (aka `{}' in base64).
	}

	auth, err := base64.StdEncoding.DecodeString(v.rawauth)
	if err != nil {
		return v, errors.Wrap(err, "could not decode params")
	}

	err = json.Unmarshal(auth, &v.auth)
	if err != nil {
		return v, errors.Wrap(err, "could not parse params")
	}

	return v, nil
}

////
///
//

func (v *vault4) seal(keychain *KeyChain, payload []byte) error {
	dek, err := hex.DecodeString(keychain.MasterKey)
	if err != nil {
		return errors.Wrap(err, "could not decode EK")
	}

	auth := v.auth.toSortedKeysJSON()
	v.rawauth = base64.StdEncoding.EncodeToString(auth)

	//
	// Encrypting

	nonce, err := GenerateRandomBytes(24)
	if err != nil {
		return errors.Wrap(err, "could not generate nonce")
	}

	aead, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return errors.Wrap(err, "could not create cipher")
	}

	ciphertext := aead.Seal(nil, nonce, payload, []byte(v.rawauth))

	//
	// Encoding

	v.nonce = hex.EncodeToString(nonce)
	v.ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	return nil
}

////
///
//

// EncryptionKey & AuthKey
func (v *vault4) unseal(keychain *KeyChain) ([]byte, error) {
	//
	// Decoding

	dek, err := hex.DecodeString(keychain.MasterKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode EK")
	}

	nonce, err := hex.DecodeString(v.nonce)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode nonce")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(v.ciphertext)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode ciphertext")
	}

	//
	// Decrypting

	aead, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return nil, errors.Wrap(err, "could not create cipher")
	}

	ciphertext, err = aead.Open(nil, nonce, ciphertext, []byte(v.rawauth))
	if err != nil {
		return nil, errors.Wrap(err, "could not decrypt")
	}

	return ciphertext, nil
}

////
///
//

func (v *vault4) setup(i *Item, old vault) {
	v.auth.KeyParams = *i.AuthParams.(*auth)
	v.auth.from = i.ContentType

	if vault, ok := old.(*vault4); ok {
		// Forward additional data from vault used to unseal to the new one created for sealing.
		v.additionaldata = vault.additionaldata
	}
}

func (v *vault4) configure(i *Item) {
	i.Version = v.version
	i.AuthParams = &v.auth.KeyParams
}

////
///
//

func (v *vault4) serialize() (string, error) {
	payload := fmt.Sprintf("%s:%s:%s:%s", v.version, v.nonce, v.ciphertext, v.rawauth)
	if v.additionaldata != "" {
		payload = fmt.Sprintf("%s:%s", payload, v.additionaldata)
	}
	return payload, nil
}

////
///
//

func (d *authenticatedData) toSortedKeysJSON() []byte {
	values := []string{}
	if d.from == ContentTypeItemsKey {
		auth := []string{}
		auth = append(auth, fmt.Sprintf(`"identifier":"%s"`, d.KeyParams.FieldIdentifier))
		auth = append(auth, fmt.Sprintf(`"origination":"%s"`, d.KeyParams.FieldOrigination))
		auth = append(auth, fmt.Sprintf(`"pw_nonce":"%s"`, d.KeyParams.FieldNonce))
		auth = append(auth, fmt.Sprintf(`"version":"%s"`, d.KeyParams.Version()))

		values = append(values, fmt.Sprintf(`"kp":{%s}`, strings.Join(auth, ",")))
	}
	values = append(values, fmt.Sprintf(`"u":"%s"`, d.UserID))
	values = append(values, fmt.Sprintf(`"v":"%s"`, d.Version))

	return []byte(fmt.Sprintf("{%s}", strings.Join(values, ",")))
}

////
///
//

// nolint:deadcode,unused
func kdf4s(password, salt string) ([]byte, error) {
	s, err := hex.DecodeString(salt)
	if err != nil {
		return nil, errors.Wrap(err, "salt is not an hexadecimal string")
	}

	return kdf4([]byte(password), s)
}

func kdf4(password, salt []byte) (k []byte, err error) {
	if len(salt) == 0 {
		salt, err = GenerateRandomBytes(16)
		if err != nil {
			return nil, errors.Wrap(err, "could not generate salt argon2id")
		}
	}

	return argon2.IDKey(password, salt, 5, 64<<10, 1, 64), nil
}
