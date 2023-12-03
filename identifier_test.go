package replicate_test

import (
	"testing"

	"github.com/replicate/replicate-go"
)

func TestValidWithVersion(t *testing.T) {
	identifier, err := replicate.ParseIdentifier("owner/name:abc123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if identifier.Owner != "owner" || identifier.Name != "name" || *identifier.Version != "abc123" {
		t.Errorf("Unexpected identifier: %+v", identifier)
	}
}

func TestValidWithoutVersion(t *testing.T) {
	identifier, err := replicate.ParseIdentifier("owner/name")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if identifier.Owner != "owner" || identifier.Name != "name" || identifier.Version != nil {
		t.Errorf("Unexpected identifier: %+v", identifier)
	}
}

func TestInvalid(t *testing.T) {
	_, err := replicate.ParseIdentifier("invalid")
	if err != replicate.ErrInvalidIdentifier {
		t.Errorf("Expected ErrInvalidIdentifier, got: %v", err)
	}
}

func TestEmpty(t *testing.T) {
	_, err := replicate.ParseIdentifier("/")
	if err != replicate.ErrInvalidIdentifier {
		t.Errorf("Expected ErrInvalidIdentifier, got: %v", err)
	}
}

func TestBlank(t *testing.T) {
	_, err := replicate.ParseIdentifier("")
	if err != replicate.ErrInvalidIdentifier {
		t.Errorf("Expected ErrInvalidIdentifier, got: %v", err)
	}
}
