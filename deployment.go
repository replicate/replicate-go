package replicate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Deployment struct {
	Owner          string            `json:"owner"`
	Name           string            `json:"name"`
	CurrentRelease DeploymentRelease `json:"current_release"`

	rawJSON json.RawMessage `json:"-"`
}

type DeploymentRelease struct {
	Number        int                     `json:"number"`
	Model         string                  `json:"model"`
	Version       string                  `json:"version"`
	CreatedAt     string                  `json:"created_at"`
	CreatedBy     Account                 `json:"created_by"`
	Configuration DeploymentConfiguration `json:"configuration"`
}

type DeploymentConfiguration struct {
	Hardware     string `json:"hardware"`
	MinInstances int    `json:"min_instances"`
	MaxInstances int    `json:"max_instances"`
}

func (d *Deployment) RawJSON() json.RawMessage {
	return d.rawJSON
}

var _ json.Unmarshaler = (*Deployment)(nil)

func (d *Deployment) UnmarshalJSON(data []byte) error {
	d.rawJSON = data
	type Alias Deployment
	alias := &struct{ *Alias }{Alias: (*Alias)(d)}
	return json.Unmarshal(data, alias)
}

// CreateDeploymentPrediction sends a request to the Replicate API to create a prediction using the specified deployment.
func (c *Client) CreatePredictionWithDeployment(ctx context.Context, deploymentOwner string, deploymentName string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
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
	path := fmt.Sprintf("/deployments/%s/%s/predictions", deploymentOwner, deploymentName)
	err := c.fetch(ctx, http.MethodPost, path, data, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	return prediction, nil
}

// GetDeployment retrieves the details of a specific deployment.
func (c *Client) GetDeployment(ctx context.Context, deploymentOwner string, deploymentName string) (*Deployment, error) {
	deployment := &Deployment{}
	path := fmt.Sprintf("/deployments/%s/%s", deploymentOwner, deploymentName)
	err := c.fetch(ctx, http.MethodGet, path, nil, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return deployment, nil
}

// ListDeployments retrieves a list of deployments associated with the current account.
func (c *Client) ListDeployments(ctx context.Context) (*Page[Deployment], error) {
	response := &Page[Deployment]{}
	path := "/deployments"
	err := c.fetch(ctx, http.MethodGet, path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	return response, nil
}

type CreateDeploymentOptions struct {
	Name         string `json:"name"`
	Model        string `json:"model"`
	Version      string `json:"version"`
	Hardware     string `json:"hardware"`
	MinInstances int    `json:"min_instances"`
	MaxInstances int    `json:"max_instances"`
}

// CreateDeployment creates a new deployment.
func (c *Client) CreateDeployment(ctx context.Context, options CreateDeploymentOptions) (*Deployment, error) {
	deployment := &Deployment{}
	path := "/deployments"
	err := c.fetch(ctx, http.MethodPost, path, options, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	return deployment, nil
}

type UpdateDeploymentOptions struct {
	Model        *string `json:"model,omitempty"`
	Version      *string `json:"version,omitempty"`
	Hardware     *string `json:"hardware,omitempty"`
	MinInstances *int    `json:"min_instances,omitempty"`
	MaxInstances *int    `json:"max_instances,omitempty"`
}

// UpdateDeployment updates an existing deployment.
func (c *Client) UpdateDeployment(ctx context.Context, deploymentOwner string, deploymentName string, options UpdateDeploymentOptions) (*Deployment, error) {
	deployment := &Deployment{}
	path := fmt.Sprintf("/deployments/%s/%s", deploymentOwner, deploymentName)
	err := c.fetch(ctx, http.MethodPatch, path, options, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to update deployment: %w", err)
	}

	return deployment, nil
}

// DeleteDeployment deletes an existing deployment.
func (c *Client) DeleteDeployment(ctx context.Context, deploymentOwner string, deploymentName string) error {
	path := fmt.Sprintf("/deployments/%s/%s", deploymentOwner, deploymentName)
	err := c.fetch(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}
	return nil
}
