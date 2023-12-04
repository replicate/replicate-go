package replicate

import (
	"errors"
	"fmt"
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

	var name, owner string
	var version *string

	owner = parts[0]
	name = parts[1]

	subparts := strings.Split(name, ":")
	if len(subparts) > 1 {
		name = subparts[0]
		version = &subparts[1]
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

func (i *Identifier) String() string {
	if i.Version == nil {
		return fmt.Sprintf("%s/%s", i.Owner, i.Name)
	}

	return fmt.Sprintf("%s/%s:%s", i.Owner, i.Name, *i.Version)
}
