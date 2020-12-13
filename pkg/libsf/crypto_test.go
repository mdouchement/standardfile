package libsf_test

import (
	"testing"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomBytes(t *testing.T) {
	for _, v := range []int{1, 8, 16, 32, 128, 512, 8192} {
		salt, err := libsf.GenerateRandomBytes(v)
		assert.NoError(t, err)
		assert.Equal(t, int(v), len(salt))
	}
}
