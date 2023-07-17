package libsf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/d1str0/pkcs7"
	"github.com/pkg/errors"
)

type vault3 struct {
	version    string
	auth       string // encrypted ciphertext hmac
	uuid       string // Item ID
	iv         string // AES's iv
	ciphertext string
	params     Auth // AuthParams json+base64 encoded
}

////
///
//

func parse3(components []string, id string) (vault, error) {
	v := &vault3{}

	if len(components) < 5 || len(components) > 6 {
		return v, errors.New("invalid secret format")
	}

	v.version = components[0]
	v.auth = components[1]
	v.uuid = components[2]
	if v.uuid != id {
		return v, errors.New("missmatch between key params UUID and item UUID")
	}

	v.iv = components[3]
	v.ciphertext = components[4]

	if len(components) == 6 {
		params, err := base64.StdEncoding.DecodeString(components[5])
		if err != nil {
			return v, errors.Wrap(err, "could not decode params")
		}

		var a auth
		err = json.Unmarshal(params, &a)
		if err != nil {
			return v, errors.Wrap(err, "could not parse params")
		}
		v.params = &a
	}

	return v, nil
}

////
///
//

func (v *vault3) seal(keychain *KeyChain, payload []byte) error {
	dek, err := hex.DecodeString(keychain.MasterKey)
	if err != nil {
		return errors.Wrap(err, "could not decode EK")
	}

	//
	// Encrypting

	block, err := aes.NewCipher(dek)
	if err != nil {
		return errors.Wrap(err, "could not create cipher")
	}

	ciphertext, err := pkcs7.Pad(payload, block.BlockSize())
	if err != nil {
		return errors.Wrap(err, "could not pkcs7 pad ciphertext")
	}

	div, err := GenerateRandomBytes(block.BlockSize())
	if err != nil {
		return errors.Wrap(err, "could not generate IV")
	}

	mode := cipher.NewCBCEncrypter(block, div)
	mode.CryptBlocks(ciphertext, ciphertext)

	//
	// Encoding

	v.iv = hex.EncodeToString(div)
	v.ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	v.auth, err = v.computeAuth(keychain.AuthKey)
	if err != nil {
		return errors.Wrap(err, "authenticate")
	}
	return nil
}

////
///
//

// EncryptionKey & AuthKey
func (v *vault3) unseal(keychain *KeyChain) ([]byte, error) {
	localAuth, err := v.computeAuth(keychain.AuthKey)
	if err != nil {
		return nil, errors.Wrap(err, "authenticate")
	}

	if localAuth != v.auth {
		return nil, errors.New("hash does not match")
	}

	//
	// Decoding

	dek, err := hex.DecodeString(keychain.MasterKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode EK")
	}

	div, err := hex.DecodeString(v.iv)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode IV")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(v.ciphertext)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode ciphertext")
	}

	//
	// Decrypting

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, errors.Wrap(err, "could not create cipher")
	}

	mode := cipher.NewCBCDecrypter(block, div)
	mode.CryptBlocks(ciphertext, ciphertext)

	ciphertext, err = pkcs7.Unpad(ciphertext)
	if err != nil {
		return nil, errors.Wrap(err, "could not pkcs7 unpad ciphertext")
	}

	return ciphertext, nil
}

////
///
//

func (v *vault3) setup(i *Item, _ vault) {
	v.params = i.AuthParams
}

func (v *vault3) configure(i *Item) {
	i.Version = v.version
	i.AuthParams = v.params
}

////
///
//

func (v *vault3) computeAuth(ak string) (string, error) {
	dak, err := hex.DecodeString(ak)
	if err != nil {
		return "", errors.Wrap(err, "could not decode AK")
	}

	ciphertextToAuth := fmt.Sprintf("%s:%s:%s:%s", v.version, v.uuid, v.iv, v.ciphertext)

	mac := hmac.New(sha256.New, dak)
	if _, err = mac.Write([]byte(ciphertextToAuth)); err != nil {
		return "", errors.Wrap(err, "could not hmac256")
	}

	return hex.EncodeToString(mac.Sum(nil)), nil
}

////
///
//

func (v *vault3) serialize() (string, error) {
	if v.params == nil {
		return fmt.Sprintf("%s:%s:%s:%s:%s", v.version, v.auth, v.uuid, v.iv, v.ciphertext), nil
	}

	a, err := json.Marshal(v.params)
	if err != nil {
		return "", errors.Wrap(err, "could not serialize params")
	}

	return fmt.Sprintf("%s:%s:%s:%s:%s:%s", v.version, v.auth, v.uuid, v.iv, v.ciphertext, base64.StdEncoding.EncodeToString(a)), nil
}
