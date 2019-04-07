package sferror_test

import (
	"testing"

	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/stretchr/testify/assert"
)

func TestSFError(t *testing.T) {
	err := sferror.New("some message")

	assert.Equal(t, "some message", err.Error())
}
