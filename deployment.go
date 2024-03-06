package replicate

import (
	"context"
	"encoding/json"
	"fmt"
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

func (d Deployment) MarshalJSON() ([]byte, error) {
	if d.rawJSON != nil {
		return d.rawJSON, nil
	} else {
		type Alias Deployment
		return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(&d)})
	}
}

func (d *Deployment) UnmarshalJSON(data []byte) error {
	d.rawJSON = data
	type Alias Deployment
	alias := &struct{ *Alias }{Alias: (*Alias)(d)}
	return json.Unmarshal(data, alias)
}

// GetDeployment retrieves the details of a specific deployment.
func (r *Client) GetDeployment(ctx context.Context, deployment_owner string, deployment_name string) (*Deployment, error) {
	deployment := &Deployment{}
	path := fmt.Sprintf("/deployments/%s/%s", deployment_owner, deployment_name)
	err := r.fetch(ctx, "GET", path, nil, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return deployment, nil
}

// CreateDeploymentPrediction sends a request to the Replicate API to create a prediction using the specified deployment.
func (r *Client) CreatePredictionWithDeployment(ctx context.Context, deploymentOwner string, deploymentName string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
	opts := []CreatePredictionOption{
		WithDeployment(deploymentOwner, deploymentName),
		WithInput(input),
		WithWebhook(webhook),
		WithStream(stream),
	}
	return r.CreatePredictionWithOptions(ctx, opts...)
}
