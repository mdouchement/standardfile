package libsf_test

import (
	"testing"
	"time"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestUnixMillisecond(t *testing.T) {
	now := time.Now()
	assert.Equal(t, now.UnixNano()/int64(time.Millisecond), libsf.UnixMillisecond(now))
}

func TestFromUnixMillisecond(t *testing.T) {
	now := time.Now()
	mnow := now.UnixNano() / int64(time.Millisecond)

	assert.Equal(t, time.Unix(0, mnow*int64(time.Millisecond)), libsf.FromUnixMillisecond(mnow))
}
