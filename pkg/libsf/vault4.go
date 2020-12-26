package libsf

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

type (
	vault4 struct {
		version    string
		nonce      string
		ciphertext string
		rawauth    string
		auth       authenticatedData
	}

	authenticatedData struct {
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

	if len(components) != 4 {
		return v, errors.New("invalid secret format")
	}

	v.version = components[0]
	v.nonce = components[1]
	v.ciphertext = components[2]
	v.rawauth = components[3]

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

	auth, err := json.Marshal(v.auth)
	if err != nil {
		return errors.Wrap(err, "could not serialize auth")
	}
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

func (v *vault4) setup(i *Item) {
	v.auth.KeyParams = *i.AuthParams.(*auth)
}

func (v *vault4) configure(i *Item) {
	i.Version = v.version
	if i.ContentType != ContentTypeItemsKey {
		i.AuthParams = &v.auth.KeyParams
	}
}

////
///
//

func (v *vault4) serialize() (string, error) {
	return fmt.Sprintf("%s:%s:%s:%s", v.version, v.nonce, v.ciphertext, v.rawauth), nil
}

////
///
//

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
