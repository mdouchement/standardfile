package libsf_test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestTimeFromToken(t *testing.T) {
	now := time.Now()
	pretoken := fmt.Sprintf("2:%d", now.UnixNano())
	token := base64.URLEncoding.EncodeToString([]byte(pretoken))

	assert.WithinDuration(t, now, libsf.TimeFromToken(token), 1*time.Nanosecond)
}

func TestTokenFromTime(t *testing.T) {
	now := time.Now()
	pretoken := fmt.Sprintf("2:%d", now.UnixNano())
	token := base64.URLEncoding.EncodeToString([]byte(pretoken))

	assert.Equal(t, token, libsf.TokenFromTime(now))
}
