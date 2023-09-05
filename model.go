package replicate

import (
	"context"
	"encoding/json"
	"fmt"
)

type Model struct {
	URL            string        `json:"url"`
	Owner          string        `json:"owner"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Visibility     string        `json:"visibility"`
	GithubURL      string        `json:"github_url"`
	PaperURL       string        `json:"paper_url"`
	LicenseURL     string        `json:"license_url"`
	RunCount       int           `json:"run_count"`
	CoverImageURL  string        `json:"cover_image_url"`
	DefaultExample *Prediction   `json:"default_example"`
	LatestVersion  *ModelVersion `json:"latest_version"`

	rawJSON json.RawMessage `json:"-"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	if m.rawJSON != nil {
		return m.rawJSON, nil
	} else {
		type Alias Model
		return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(&m)})
	}
}

func (m *Model) UnmarshalJSON(data []byte) error {
	m.rawJSON = data
	type Alias Model
	alias := &struct{ *Alias }{Alias: (*Alias)(m)}
	return json.Unmarshal(data, alias)
}

type ModelVersion struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	CogVersion    string      `json:"cog_version"`
	OpenAPISchema interface{} `json:"openapi_schema"`

	rawJSON json.RawMessage `json:"-"`
}

func (m ModelVersion) MarshalJSON() ([]byte, error) {
	if m.rawJSON != nil {
		return m.rawJSON, nil
	} else {
		type Alias ModelVersion
		return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(&m)})
	}
}

func (m *ModelVersion) UnmarshalJSON(data []byte) error {
	m.rawJSON = data
	type Alias ModelVersion
	alias := &struct{ *Alias }{Alias: (*Alias)(m)}
	return json.Unmarshal(data, alias)
}

// GetModel retrieves information about a model.
func (r *Client) GetModel(ctx context.Context, modelOwner string, modelName string) (*Model, error) {
	model := &Model{}
	err := r.request(ctx, "GET", fmt.Sprintf("/models/%s/%s", modelOwner, modelName), nil, model)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return model, nil
}

// ListModelVersions lists the versions of a model.
func (r *Client) ListModelVersions(ctx context.Context, modelOwner string, modelName string) (*Page[ModelVersion], error) {
	response := &Page[ModelVersion]{}
	err := r.request(ctx, "GET", fmt.Sprintf("/models/%s/%s/versions", modelOwner, modelName), nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list model versions: %w", err)
	}
	return response, nil
}

// GetModelVersion retrieves a specific version of a model.
func (r *Client) GetModelVersion(ctx context.Context, modelOwner string, modelName string, versionID string) (*ModelVersion, error) {
	version := &ModelVersion{}
	err := r.request(ctx, "GET", fmt.Sprintf("/models/%s/%s/versions/%s", modelOwner, modelName, versionID), nil, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get model version: %w", err)
	}
	return version, nil
}
