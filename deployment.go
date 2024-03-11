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

func (d Deployment) MarshalJSON() ([]byte, error) {
	if d.rawJSON != nil {
		return d.rawJSON, nil
	}
	type Alias Deployment
	return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(&d)})
}

func (d *Deployment) UnmarshalJSON(data []byte) error {
	d.rawJSON = data
	type Alias Deployment
	alias := &struct{ *Alias }{Alias: (*Alias)(d)}
	return json.Unmarshal(data, alias)
}

// GetDeployment retrieves the details of a specific deployment.
func (r *Client) GetDeployment(ctx context.Context, deploymentOwner string, deploymentName string) (*Deployment, error) {
	deployment := &Deployment{}
	path := fmt.Sprintf("/deployments/%s/%s", deploymentOwner, deploymentName)
	err := r.fetch(ctx, http.MethodGet, path, nil, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return deployment, nil
}

// CreateDeploymentPrediction sends a request to the Replicate API to create a prediction using the specified deployment.
func (r *Client) CreatePredictionWithDeployment(ctx context.Context, deploymentOwner string, deploymentName string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
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
	err := r.fetch(ctx, http.MethodPost, path, data, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	return prediction, nil
}
