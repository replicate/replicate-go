package replicate

import (
	"context"
	"fmt"
)

type Model struct {
	URL            string        `json:"url"`
	Owner          string        `json:"owner"`
	Name           string        `json:"name"`
	Description    *string       `json:"description,omitempty"`
	Visibility     string        `json:"visibility"`
	GithubURL      *string       `json:"github_url,omitempty"`
	PaperURL       *string       `json:"paper_url,omitempty"`
	LicenseURL     *string       `json:"license_url,omitempty"`
	RunCount       int           `json:"run_count"`
	CoverImageURL  *string       `json:"cover_image_url,omitempty"`
	DefaultExample *Prediction   `json:"default_example,omitempty"`
	LatestVersion  *ModelVersion `json:"latest_version,omitempty"`
}

type ModelVersion struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	CogVersion    string      `json:"cog_version"`
	OpenAPISchema interface{} `json:"openapi_schema"`
}

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
