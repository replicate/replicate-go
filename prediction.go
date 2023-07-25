package replicate

import (
	"context"
	"errors"
	"fmt"
	"time"
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
	StartedAt           *string            `json:"started_at,omitempty"`
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

// ListPredictions returns a paginated list of predictions.
func (r *Client) ListPredictions(ctx context.Context) (*Page[Prediction], error) {
	response := &Page[Prediction]{}
	err := r.request(ctx, "GET", "/predictions", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	return response, nil
}

// GetPrediction retrieves a prediction from the Replicate API by its ID.
func (r *Client) GetPrediction(ctx context.Context, id string) (*Prediction, error) {
	prediction := &Prediction{}
	err := r.request(ctx, "GET", fmt.Sprintf("/predictions/%s", id), nil, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to get prediction: %w", err)
	}
	return prediction, nil
}

// Wait for a prediction to finish.
//
// This function blocks until the prediction has finished, or the context is cancelled.
// If the prediction has already finished, the prediction is returned immediately.
// If the prediction has not finished after maxAttempts, an error is returned.
// If interval is 1, the prediction is checked only once.
// If interval is negative, an error is returned.
// If maxAttempts is 0, there is no limit to the number of attempts.
// If maxAttempts is negative, an error is returned.
func (r *Client) Wait(ctx context.Context, prediction Prediction, interval time.Duration, maxAttempts int) (*Prediction, error) {
	if prediction.Status.Terminated() {
		return &prediction, nil
	}

	if interval <= 0 {
		return nil, errors.New("interval must be greater than zero")
	}

	if maxAttempts < 0 {
		return nil, errors.New("maxAttempts must be greater than or equal to zero")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	id := prediction.ID

	attempts := 0
	for {
		select {
		case <-ticker.C:
			prediction, err := r.GetPrediction(ctx, id)
			if err != nil {
				return nil, err
			}

			if prediction.Status.Terminated() {
				return prediction, nil
			}

			attempts += 1
			if maxAttempts > 0 && attempts > maxAttempts {
				return nil, fmt.Errorf("prediction %s did not finish after %d attempts", id, maxAttempts)
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
