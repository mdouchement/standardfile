package service

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
)

type (
	// A PKCEService is a service used for managing challenges.
	PKCEService interface {
		ComputeChallenge(codeVerifier string) string
		StoreChallenge(codeChallenge string) error
		CheckChallenge(codeChallenge string) error
	}

	pkceService struct {
		db     database.Client
		Params Params
	}
)

// NewPKCE instantiates a new PKCE service.
func NewPKCE(db database.Client, params Params) (s PKCEService) {
	switch params.APIVersion { // for future API increments
	default:
		s = &pkceService{
			db:     db,
			Params: params,
		}
	}
	return s
}

func (s *pkceService) ComputeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	hexHash := fmt.Sprintf("%x", hash)
	return base64.RawURLEncoding.EncodeToString([]byte(hexHash))
}

func (s *pkceService) StoreChallenge(codeChallenge string) error {
	if err := s.db.RevokeExpiredChallenges(); err != nil {
		return err
	}

	return s.db.Save(&model.PKCE{
		CodeChallenge: codeChallenge,
		ExpireAt:      time.Now().Add(1 * time.Hour).UTC(),
	})
}

func (s *pkceService) CheckChallenge(codeChallenge string) error {
	if err := s.db.RevokeExpiredChallenges(); err != nil {
		return err
	}

	if _, err := s.db.FindPKCE(codeChallenge); err != nil {
		return err
	}

	return s.db.RemovePKCE(codeChallenge)
}
