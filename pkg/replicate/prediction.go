package replicate

import (
	"context"
	"fmt"
)

type Source string

const (
	SourceWeb Source = "web"
	SourceAPI Source = "api"
)

type Prediction struct {
	ID      string           `json:"id"`
	Status  Status           `json:"status"`
	Version string           `json:"version"`
	Input   PredictionInput  `json:"input"`
	Output  PredictionOutput `json:"output,omitempty"`
	Source  Source           `json:"source"`
	Error   interface{}      `json:"error,omitempty"`
	Logs    *string          `json:"logs,omitempty"`
	Metrics *struct {
		PredictTime *float64 `json:"predict_time,omitempty"`
	} `json:"metrics,omitempty"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
	CreatedAt           string             `json:"created_at"`
	UpdatedAt           string             `json:"updated_at"`
	CompletedAt         *string            `json:"completed_at,omitempty"`
}

type PredictionInput map[string]interface{}
type PredictionOutput interface{}

// CreatePrediction sends a request to the Replicate API to create a prediction.
func (r *Client) CreatePrediction(ctx context.Context, version string, input PredictionInput, webhook *Webhook) (*Prediction, error) {
	data := map[string]interface{}{
		"version": version,
		"input":   input,
	}

	if webhook != nil {
		data["webhook"] = webhook.URL
		data["webhook_events_filter"] = webhook.Events
	}

	prediction := &Prediction{}
	err := r.request(ctx, "POST", "/predictions", data, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	return prediction, nil
}