package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/chzyer/readline"
	sargon2 "github.com/mdouchement/simple-argon2"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	saltKeyLength   = 16
	credentialsfile = ".standardfile"
)

// A Config holds client's configuration.
type Config struct {
	Endpoint    string         `json:"endpoint"`
	Email       string         `json:"email"`
	BearerToken string         `json:"bearer_token"` // JWT used before 20200115
	Session     libsf.Session  `json:"session"`      // Since 20200115
	KeyChain    libsf.KeyChain `json:"keychain"`
}

// Remove removes the credential files from the current directory.
func Remove() error {
	return os.Remove(credentialsfile)
}

// Load gets the configuration from the current folder according to `credentialsfile` const.
func Load() (Config, error) {
	fmt.Println("Loading credentials from " + credentialsfile)
	cfg := Config{
		KeyChain: libsf.KeyChain{
			ItemsKey: make(map[string]string),
		},
	}

	ciphertext, err := ioutil.ReadFile(credentialsfile)
	if err != nil {
		return cfg, errors.Wrap(err, "could not read credentials file")
	}

	//
	// Key derivation of passphrase

	passphrase, err := readline.Password("passphrase: ")
	if err != nil {
		return cfg, errors.Wrap(err, "could not read passphrase from stdin")
	}

	salt := ciphertext[:saltKeyLength]
	ciphertext = ciphertext[saltKeyLength:]
	hash := argon2.IDKey(passphrase, salt, 3, 64<<10, 2, 32)

	//
	// Seal config

	aead, err := chacha20poly1305.NewX(hash)
	if err != nil {
		return cfg, errors.Wrap(err, "could not create AEAD")
	}

	nonce := ciphertext[:aead.NonceSize()]
	ciphertext = ciphertext[aead.NonceSize():]

	payload, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return cfg, errors.Wrap(err, "could not decrypt credentials file")
	}

	err = json.Unmarshal(payload, &cfg)
	if err != nil {
		return cfg, errors.Wrap(err, "could not parse config")
	}

	return cfg, nil
}

// Save stores the configuration in the current folder according to `credentialsfile` const.
func Save(cfg Config) error {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "could not serialize config")
	}

	fmt.Println("Storing credentials in current directory as " + credentialsfile)
	passphrase, err := readline.Password("passphrase: ")
	if err != nil {
		return errors.Wrap(err, "could not read passphrase from stdin")
	}

	//
	// Key derivation of passphrase

	salt, err := sargon2.GenerateRandomBytes(saltKeyLength)
	if err != nil {
		return errors.Wrap(err, "could not generate salt for credentials")
	}
	hash := argon2.IDKey(passphrase, salt, 3, 64<<10, 2, 32)

	//
	// Seal config

	aead, err := chacha20poly1305.NewX(hash)
	if err != nil {
		return errors.Wrap(err, "could not create AEAD")
	}
	nonce, err := sargon2.GenerateRandomBytes(uint32(aead.NonceSize()))
	if err != nil {
		return errors.Wrap(err, "could not generate nonce for credentials")
	}

	ciphertext := aead.Seal(nil, nonce, payload, nil)
	ciphertext = append(nonce, ciphertext...)
	ciphertext = append(salt, ciphertext...)

	f, err := os.Create(credentialsfile)
	if err != nil {
		return errors.Wrapf(err, "could not create %s", credentialsfile)
	}
	defer f.Close()

	_, err = f.Write(ciphertext)
	if err != nil {
		return errors.Wrap(err, "could not store credentials")
	}

	return errors.Wrap(f.Sync(), "could not store credentials")
}

// Refresh refreshes the session if needed.
func Refresh(client libsf.Client, cfg *Config) error {
	if !cfg.Session.AccessExpiredAt(time.Now().Add(time.Hour)) {
		return nil
	}

	fmt.Println("Refreshing the session")

	session, err := client.RefreshSession(cfg.Session.AccessToken, cfg.Session.RefreshToken)
	if err != nil {
		return errors.Wrap(err, "could not refresh session")
	}
	cfg.Session = *session
	client.SetSession(cfg.Session)

	err = Save(*cfg)
	return errors.Wrap(err, "could not save refreshed session")
}
