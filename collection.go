package replicate

import (
	"context"
	"fmt"
)

type Collection struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Models      *[]Model `json:"models,omitempty"`
}

// ListCollections returns a list of all collections.
func (r *Client) ListCollections(ctx context.Context) (*Page[Collection], error) {
	response := &Page[Collection]{}
	err := r.request(ctx, "GET", "/collections", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return response, nil
}

// GetCollection returns a collection by slug.
func (r *Client) GetCollection(ctx context.Context, slug string) (*Collection, error) {
	collection := &Collection{}
	err := r.request(ctx, "GET", fmt.Sprintf("/collections/%s", slug), nil, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return collection, nil
}
