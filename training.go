package replicate

import (
	"context"
	"fmt"
)

type Training Prediction
type TrainingInput PredictionInput

type createTrainingBody struct {
	Destination         *string            `json:"destination,omitempty"`
	Input               TrainingInput      `json:"input"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
}

type CreateTrainingOption interface {
	applyToCreateTrainingPath(*string)
	applyToCreateTrainingBody(*createTrainingBody)
}

var _ CreateTrainingOption = withDestinationOption{}
var _ CreateTrainingOption = withModelVersionOption{}
var _ CreateTrainingOption = withInputOption{}
var _ CreateTrainingOption = withWebhookOption{}

//

type withDestinationOption struct {
	destination string
}

func (o withDestinationOption) applyToCreateTrainingPath(path *string) {}
func (o withDestinationOption) applyToCreateTrainingBody(body *createTrainingBody) {
	body.Destination = &o.destination
}

func WithDestination(destination string) CreateTrainingOption {
	return withDestinationOption{destination: destination}
}

//

func (o withModelVersionOption) applyToCreateTrainingPath(path *string) {
	(*path) = fmt.Sprintf("/models/%s/%s/versions/%s/trainings", o.owner, o.name, o.version)
}
func (o withModelVersionOption) applyToCreateTrainingBody(body *createTrainingBody) {}

func (o withInputOption) applyToCreateTrainingPath(path *string) {}
func (o withInputOption) applyToCreateTrainingBody(body *createTrainingBody) {
	body.Input = o.input
}

func (o withWebhookOption) applyToCreateTrainingPath(path *string) {}
func (o withWebhookOption) applyToCreateTrainingBody(body *createTrainingBody) {
	if o.webhook != nil {
		body.Webhook = &o.webhook.URL
		if len(o.webhook.Events) > 0 {
			body.WebhookEventsFilter = o.webhook.Events
		}
	}
}

func (r *Client) CreateTrainingWithOptions(ctx context.Context, opts ...CreateTrainingOption) (*Training, error) {
	path := "/trainings"
	body := &createTrainingBody{}
	for _, opt := range opts {
		opt.applyToCreateTrainingPath(&path)
		opt.applyToCreateTrainingBody(body)
	}

	training := &Training{}
	err := r.fetch(ctx, "POST", path, body, training)
	if err != nil {
		return nil, fmt.Errorf("failed to create training: %w", err)
	}

	return training, nil
}

// CreateTraining sends a request to the Replicate API to create a new training.
func (r *Client) CreateTraining(ctx context.Context, modelOwner string, modelName string, version string, destination string, input TrainingInput, webhook *Webhook) (*Training, error) {
	opts := []CreateTrainingOption{
		WithModelVersion(modelOwner, modelName, version),
		WithDestination(destination),
		WithInput(input),
		WithWebhook(webhook),
	}
	return r.CreateTrainingWithOptions(ctx, opts...)
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
