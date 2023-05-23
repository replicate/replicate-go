package replicate

import (
	"context"
	"fmt"
)

type Training Prediction

// CreatePrediction sends a request to the Replicate API to create a prediction.
func (r *Client) CreateTraining(ctx context.Context, model_owner string, model_name string, version string, destination string, input PredictionInput, webhook *Webhook) (*Training, error) {
	data := map[string]interface{}{
		"version":     version,
		"destination": destination,
		"input":       input,
	}

	if webhook != nil {
		data["webhook"] = webhook.URL
		data["webhook_events_filter"] = webhook.Events
	}

	training := &Training{}
	path := fmt.Sprintf("/models/%s/%s/versions/%s/trainings", model_owner, model_name, version)
	err := r.request(ctx, "POST", path, data, training)
	if err != nil {
		return nil, fmt.Errorf("failed to create training: %w", err)
	}

	return training, nil
}
