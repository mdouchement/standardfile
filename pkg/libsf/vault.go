package libsf

import (
	"strings"

	"github.com/pkg/errors"
)

type vault interface {
	setup(i *Item, old vault)
	seal(keychain *KeyChain, payload []byte) error
	serialize() (string, error)
	unseal(keychain *KeyChain) ([]byte, error)
	configure(i *Item)
}

////
///
//

func create(version, id string) (vault, error) {
	switch version {
	case ProtocolVersion2:
		fallthrough
	case ProtocolVersion3:
		return &vault3{
			version: version,
			uuid:    id,
		}, nil
	case ProtocolVersion4:
		return &vault4{
			version: version,
			auth: authenticatedData{
				Version: version,
				UserID:  id,
			},
		}, nil
	default:
		return nil, errors.New("unsupported secret version")
	}
}

////
///
//

func parse(secret, id string) (vault, error) {
	components := strings.Split(secret, ":")

	if len(components) == 0 {
		return nil, errors.New("invalid secret format/length")
	}

	switch components[0] {
	case ProtocolVersion2:
		fallthrough
	case ProtocolVersion3:
		return parse3(components, id)
	case ProtocolVersion4:
		return parse4(components)
	default:
		return nil, errors.New("unsupported secret version")
	}
}
