package replicate

import (
	"context"
	"fmt"
)

// CreateDeploymentPrediction sends a request to the Replicate API to create a prediction using the specified deployment.
func (r *Client) CreatePredictionWithDeployment(ctx context.Context, deployment_owner string, deployment_name string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
	data := map[string]interface{}{
		"input": input,
	}

	if webhook != nil {
		data["webhook"] = webhook.URL
		data["webhook_events_filter"] = webhook.Events
	}

	if stream {
		data["stream"] = true
	}

	prediction := &Prediction{}
	path := fmt.Sprintf("/deployments/%s/%s/predictions", deployment_owner, deployment_name)
	err := r.fetch(ctx, "POST", path, data, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	return prediction, nil
}
