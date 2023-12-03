package replicate_test

import (
	"testing"

	"github.com/replicate/replicate-go"
	"github.com/stretchr/testify/assert"
)

func TestValidWithVersion(t *testing.T) {
	identifier, err := replicate.ParseIdentifier("owner/name:abc123")
	assert.NoError(t, err)
	assert.Equal(t, "owner", identifier.Owner)
	assert.Equal(t, "name", identifier.Name)
	assert.Equal(t, "abc123", *identifier.Version)
}

func TestValidWithoutVersion(t *testing.T) {
	identifier, err := replicate.ParseIdentifier("owner/name")
	assert.NoError(t, err)
	assert.Equal(t, "owner", identifier.Owner)
	assert.Equal(t, "name", identifier.Name)
	assert.Nil(t, identifier.Version)
}

func TestInvalid(t *testing.T) {
	_, err := replicate.ParseIdentifier("invalid")
	assert.Equal(t, replicate.ErrInvalidIdentifier, err)
}

func TestEmpty(t *testing.T) {
	_, err := replicate.ParseIdentifier("/")
	assert.Equal(t, replicate.ErrInvalidIdentifier, err)
}

func TestBlank(t *testing.T) {
	_, err := replicate.ParseIdentifier("")
	assert.Equal(t, replicate.ErrInvalidIdentifier, err)
}
