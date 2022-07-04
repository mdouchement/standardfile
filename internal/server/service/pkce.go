package service

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/mdouchement/standardfile/internal/database"
)

type (
	// A PKCEService is a service used for managing challenges.
	PKCEService interface {
		ComputeChallenge(code_verifier string) string
		StoreChallenge(code_challenge string) error
		CheckChallenge(code_challenge string) error
	}

	pkceService struct {
		db     database.Client
		Params Params `json:"-"`
	}
)

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

func (s *pkceService) ComputeChallenge(code_verifier string) string {
	hash := sha256.Sum256([]byte(code_verifier))
	hex_hash := fmt.Sprintf("%x", hash[:])
	return base64.RawStdEncoding.EncodeToString([]byte(hex_hash))
}

func (s *pkceService) StoreChallenge(code_challenge string) error {
	if err := s.db.RevokeExpiredChallenges(); err != nil {
		return err
	}
	return s.db.StorePKCE(code_challenge)
}

func (s *pkceService) CheckChallenge(code_challenge string) error {
	if err := s.db.RevokeExpiredChallenges(); err != nil {
		return err
	}
	if err := s.db.CheckPKCE(code_challenge); err != nil {
		return err
	}
	return s.db.RemovePKCE(code_challenge)
}
