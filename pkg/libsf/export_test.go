package libsf

// This file is only for test purpose and is only loaded by test framework.

// NewAuth returns a new Auth with the given parameters for test purpose.
func NewAuth(email, version, nonce string, cost int) Auth {
	return &auth{
		FieldIdentifier: email,
		FieldVersion:    version,
		FieldCost:       cost,
		FieldNonce:      nonce,
	}
}
