package replicate

import (
	"errors"
	"strings"
)

var (
	ErrInvalidIdentifier = errors.New("invalid identifier, it must be in the format \"owner/name\" or \"owner/name:version\"")
)

// Identifier represents a reference to a Replicate model with an optional version.
type Identifier struct {
	// Owner is the username of the model owner.
	Owner string

	// Name is the name of the model.
	Name string

	// Version is the version of the model.
	Version *string
}

func ParseIdentifier(identifier string) (*Identifier, error) {
	parts := strings.Split(identifier, "/")
	if len(parts) != 2 {
		return nil, ErrInvalidIdentifier
	}

	owner := parts[0]
	nameVersion := strings.SplitN(parts[1], ":", 2)

	name := nameVersion[0]
	var version *string
	if len(nameVersion) > 1 {
		version = &nameVersion[1]
	}

	if owner == "" || name == "" {
		return nil, ErrInvalidIdentifier
	}

	return &Identifier{
		Owner:   owner,
		Name:    name,
		Version: version,
	}, nil
}