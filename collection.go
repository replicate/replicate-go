package replicate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Collection struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Models      *[]Model `json:"models,omitempty"`

	rawJSON json.RawMessage `json:"-"`
}

func (c Collection) MarshalJSON() ([]byte, error) {
	if c.rawJSON != nil {
		return c.rawJSON, nil
	}
	type Alias Collection
	return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(&c)})
}

func (c *Collection) UnmarshalJSON(data []byte) error {
	c.rawJSON = data
	type Alias Collection
	alias := &struct{ *Alias }{Alias: (*Alias)(c)}
	return json.Unmarshal(data, alias)
}

// ListCollections returns a list of all collections.
func (r *Client) ListCollections(ctx context.Context) (*Page[Collection], error) {
	response := &Page[Collection]{}
	err := r.fetch(ctx, http.MethodGet, "/collections", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return response, nil
}

// GetCollection returns a collection by slug.
func (r *Client) GetCollection(ctx context.Context, slug string) (*Collection, error) {
	collection := &Collection{}
	err := r.fetch(ctx, http.MethodGet, fmt.Sprintf("/collections/%s", slug), nil, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return collection, nil
}
