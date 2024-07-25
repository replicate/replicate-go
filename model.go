package replicate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

func (m *Model) RawJSON() json.RawMessage {
	return m.rawJSON
}

var _ json.Unmarshaler = (*Model)(nil)

func (m *Model) UnmarshalJSON(data []byte) error {
	m.rawJSON = data
	type Alias Model
	alias := &struct{ *Alias }{Alias: (*Alias)(m)}
	return json.Unmarshal(data, alias)
}

type CreateModelOptions struct {
	Visibility    string  `json:"visibility"`
	Hardware      string  `json:"hardware"`
	Description   *string `json:"description,omitempty"`
	GithubURL     *string `json:"github_url,omitempty"`
	PaperURL      *string `json:"paper_url,omitempty"`
	LicenseURL    *string `json:"license_url,omitempty"`
	CoverImageURL *string `json:"cover_image_url,omitempty"`
}

type ModelVersion struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	CogVersion    string      `json:"cog_version"`
	OpenAPISchema interface{} `json:"openapi_schema"`

	rawJSON json.RawMessage `json:"-"`
}

func (m *ModelVersion) RawJSON() json.RawMessage {
	return m.rawJSON
}

var _ json.Unmarshaler = (*ModelVersion)(nil)

func (m *ModelVersion) UnmarshalJSON(data []byte) error {
	m.rawJSON = data
	type Alias ModelVersion
	alias := &struct{ *Alias }{Alias: (*Alias)(m)}
	return json.Unmarshal(data, alias)
}

// ListModels lists public models.
func (r *Client) ListModels(ctx context.Context) (*Page[Model], error) {
	response := &Page[Model]{}
	err := r.fetch(ctx, http.MethodGet, "/models", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	return response, nil
}

// SearchModels searches for public models.
func (r *Client) SearchModels(ctx context.Context, query string) (*Page[Model], error) {
	response := &Page[Model]{}
	request, err := r.newRequest(ctx, "QUERY", "/models", strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Content-Type", "text/plain")
	err = r.do(request, response)
	if err != nil {
		return nil, fmt.Errorf("failed to search models: %w", err)
	}
	return response, nil
}

// GetModel retrieves information about a model.
func (r *Client) GetModel(ctx context.Context, modelOwner string, modelName string) (*Model, error) {
	model := &Model{}
	err := r.fetch(ctx, http.MethodGet, fmt.Sprintf("/models/%s/%s", modelOwner, modelName), nil, model)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return model, nil
}

// CreateModel creates a new model.
func (r *Client) CreateModel(ctx context.Context, modelOwner string, modelName string, options CreateModelOptions) (*Model, error) {
	model := &Model{}

	body := struct {
		Owner string `json:"owner"`
		Name  string `json:"name"`
		CreateModelOptions
	}{
		Owner:              modelOwner,
		Name:               modelName,
		CreateModelOptions: options,
	}

	err := r.fetch(ctx, http.MethodPost, "/models", body, model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}
	return model, nil
}

// DeleteModel deletes a model with no associated versions.
func (r *Client) DeleteModel(ctx context.Context, modelOwner string, modelName string) error {
	err := r.fetch(ctx, http.MethodDelete, fmt.Sprintf("/models/%s/%s", modelOwner, modelName), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// ListModelVersions lists the versions of a model.
func (r *Client) ListModelVersions(ctx context.Context, modelOwner string, modelName string) (*Page[ModelVersion], error) {
	response := &Page[ModelVersion]{}
	err := r.fetch(ctx, http.MethodGet, fmt.Sprintf("/models/%s/%s/versions", modelOwner, modelName), nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list model versions: %w", err)
	}
	return response, nil
}

// GetModelVersion retrieves a specific version of a model.
func (r *Client) GetModelVersion(ctx context.Context, modelOwner string, modelName string, versionID string) (*ModelVersion, error) {
	version := &ModelVersion{}
	err := r.fetch(ctx, http.MethodGet, fmt.Sprintf("/models/%s/%s/versions/%s", modelOwner, modelName, versionID), nil, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get model version: %w", err)
	}
	return version, nil
}

// DeleteModelVersion deletes a model version and all associated predictions, including all output files.
func (r *Client) DeleteModelVersion(ctx context.Context, modelOwner string, modelName string, versionID string) error {
	err := r.fetch(ctx, http.MethodDelete, fmt.Sprintf("/models/%s/%s/versions/%s", modelOwner, modelName, versionID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete model version: %w", err)
	}
	return nil
}

// CreatePredictionWithModel sends a request to the Replicate API to create a prediction for a model.
func (r *Client) CreatePredictionWithModel(ctx context.Context, modelOwner string, modelName string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
	data := map[string]interface{}{
		"input": input,
	}

	if webhook != nil {
		data["webhook"] = webhook.URL
		if len(webhook.Events) > 0 {
			data["webhook_events_filter"] = webhook.Events
		}
	}

	if stream {
		data["stream"] = true
	}

	prediction := &Prediction{}
	err := r.fetch(ctx, http.MethodPost, fmt.Sprintf("/models/%s/%s/predictions", modelOwner, modelName), data, prediction)
	if err != nil {
		return nil, err
	}

	return prediction, nil
}
