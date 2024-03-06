package replicate

import (
	"context"
	"fmt"
)

type Training Prediction
type TrainingInput PredictionInput

type createTrainingBody struct {
	Version             *string            `json:"version,omitempty"`
	Input               TrainingInput      `json:"input"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
}
type CreateTrainingOption func(path *string, body *createTrainingBody)

// CreateTraining sends a request to the Replicate API to create a new training.
func (r *Client) CreateTraining(ctx context.Context, model_owner string, model_name string, version string, destination string, input TrainingInput, webhook *Webhook) (*Training, error) {
	data := map[string]interface{}{
		"version":     version,
		"destination": destination,
		"input":       input,
	}

	if webhook != nil {
		data["webhook"] = webhook.URL
		if len(webhook.Events) > 0 {
			data["webhook_events_filter"] = webhook.Events
		}
	}

	training := &Training{}
	path := fmt.Sprintf("/models/%s/%s/versions/%s/trainings", model_owner, model_name, version)
	err := r.fetch(ctx, "POST", path, data, training)
	if err != nil {
		return nil, fmt.Errorf("failed to create training: %w", err)
	}

	return training, nil
}

// ListTrainings returns a list of trainings.
func (r *Client) ListTrainings(ctx context.Context) (*Page[Training], error) {
	response := &Page[Training]{}
	err := r.fetch(ctx, "GET", "/trainings", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list trainings: %w", err)
	}
	return response, nil
}

// GetTraining sends a request to the Replicate API to get a training.
func (r *Client) GetTraining(ctx context.Context, trainingID string) (*Training, error) {
	training := &Training{}
	err := r.fetch(ctx, "GET", fmt.Sprintf("/trainings/%s", trainingID), nil, training)
	if err != nil {
		return nil, fmt.Errorf("failed to get training: %w", err)
	}

	return training, nil
}

// CancelTraining sends a request to the Replicate API to cancel a training.
func (r *Client) CancelTraining(ctx context.Context, trainingID string) (*Training, error) {
	training := &Training{}
	err := r.fetch(ctx, "POST", fmt.Sprintf("/trainings/%s/cancel", trainingID), nil, training)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel training: %w", err)
	}

	return training, nil
}
