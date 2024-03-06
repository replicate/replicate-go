package replicate

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Source string

const (
	SourceWeb Source = "web"
	SourceAPI Source = "api"
)

type Prediction struct {
	ID      string           `json:"id"`
	Status  Status           `json:"status"`
	Model   string           `json:"model"`
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
	URLs                map[string]string  `json:"urls,omitempty"`
	CreatedAt           string             `json:"created_at"`
	StartedAt           *string            `json:"started_at,omitempty"`
	CompletedAt         *string            `json:"completed_at,omitempty"`

	rawJSON json.RawMessage `json:"-"`
}

func (p Prediction) MarshalJSON() ([]byte, error) {
	if p.rawJSON != nil {
		return p.rawJSON, nil
	} else {
		type Alias Prediction
		return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(&p)})
	}
}

func (p *Prediction) UnmarshalJSON(data []byte) error {
	p.rawJSON = data
	type Alias Prediction
	alias := &struct{ *Alias }{Alias: (*Alias)(p)}
	return json.Unmarshal(data, alias)
}

type PredictionProgress struct {
	Percentage float64
	Current    int
	Total      int
}

func (p Prediction) Progress() *PredictionProgress {
	if p.Logs == nil || *p.Logs == "" {
		return nil
	}

	pattern := `^\s*(?P<percentage>\d+)%\s*\|.+?\|\s*(?P<current>\d+)\/(?P<total>\d+)`
	re := regexp.MustCompile(pattern)

	lines := strings.Split(*p.Logs, "\n")
	if len(lines) == 0 {
		return nil
	}

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if re.MatchString(line) {
			matches := re.FindStringSubmatch(lines[i])
			if len(matches) == 4 {
				var percentage, current, total int
				fmt.Sscanf(matches[1], "%d", &percentage)
				fmt.Sscanf(matches[2], "%d", &current)
				fmt.Sscanf(matches[3], "%d", &total)
				return &PredictionProgress{
					Percentage: float64(percentage) / float64(100),
					Current:    current,
					Total:      total,
				}
			}
		}
	}

	return nil
}

type PredictionInput map[string]interface{}
type PredictionOutput interface{}

type withModelSetter interface {
	setModel(owner string, name string)
}

func WithModel[T withModelSetter](owner string, name string) func(T) {
	return func(req T) {
		req.setModel(owner, name)
	}
}

type withModelVersionSetter interface {
	setModelVersion(owner string, name string, version string)
}

func WithModelVersion[T withModelVersionSetter](owner string, name string, version string) func(T) {
	return func(req T) {
		req.setModelVersion(owner, name, version)
	}
}

type withVersionSetter interface {
	setVersion(version *string)
}

func WithVersion[T withVersionSetter](version string) func(T) {
	return func(req T) {
		req.setVersion(&version)
	}
}

type withDeploymentSetter interface {
	setDeployment(owner string, name string)
}

func WithDeployment[T withDeploymentSetter](owner string, name string) func(T) {
	return func(req T) {
		req.setDeployment(owner, name)
	}
}

type withInputSetter interface {
	setInput(input PredictionInput)
}

func WithInput[T withInputSetter](input PredictionInput) func(T) {
	return func(req T) {
		req.setInput(input)
	}
}

type withWebhookSetter interface {
	setWebhook(webhook *Webhook)
}

func WithWebhook[T withWebhookSetter](webhook *Webhook) func(T) {
	return func(req T) {
		req.setWebhook(webhook)
	}
}

type withStreamSetter interface {
	setStream(stream bool)
}

func WithStream[T withStreamSetter](stream bool) func(T) {
	return func(req T) {
		req.setStream(stream)
	}
}

type createPredictionBody struct {
	Version             *string            `json:"version,omitempty"`
	Input               PredictionInput    `json:"input"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
	Stream              *bool              `json:"stream,omitempty"`
}
type createPredictionRequest struct {
	path string
	body createPredictionBody
}

var _ withModelSetter = &createPredictionRequest{}
var _ withModelVersionSetter = &createPredictionRequest{}
var _ withVersionSetter = &createPredictionRequest{}
var _ withDeploymentSetter = &createPredictionRequest{}
var _ withInputSetter = &createPredictionRequest{}
var _ withWebhookSetter = &createPredictionRequest{}
var _ withStreamSetter = &createPredictionRequest{}

func (c *createPredictionRequest) setModel(owner string, name string) {
	c.path = fmt.Sprintf("/models/%s/%s/predictions", owner, name)
	c.body.Version = nil
}

func (c *createPredictionRequest) setModelVersion(owner string, name string, version string) {
	c.path = fmt.Sprintf("/models/%s/%s/versions/%s/predictions", owner, name, version)
	c.body.Version = nil
}

func (c *createPredictionRequest) setVersion(version *string) {
	c.path = "/predictions"
	c.body.Version = version
}

func (c *createPredictionRequest) setDeployment(owner string, name string) {
	c.path = fmt.Sprintf("/deployments/%s/%s/predictions", owner, name)
	c.body.Version = nil
}

func (c *createPredictionRequest) setInput(input PredictionInput) {
	c.body.Input = input
}

func (c *createPredictionRequest) setWebhook(webhook *Webhook) {
	c.body.Webhook = &webhook.URL
	if len(webhook.Events) > 0 {
		c.body.WebhookEventsFilter = webhook.Events
	}
}

func (c *createPredictionRequest) setStream(stream bool) {
	c.body.Stream = &stream
}

type CreatePredictionOption func(*createPredictionRequest)

func (r *Client) CreatePredictionWithOptions(ctx context.Context, opts ...func(*createPredictionRequest)) (*Prediction, error) {
	req := &createPredictionRequest{}
	for _, opt := range opts {
		opt(req)
	}

	prediction := &Prediction{}
	err := r.fetch(ctx, "POST", req.path, req.body, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	return prediction, nil
}

// CreatePrediction sends a request to the Replicate API to create a prediction.
func (r *Client) CreatePrediction(ctx context.Context, version string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
	opts := []CreatePredictionOption{
		WithVersion(version),
		WithInput(input),
		WithWebhook(webhook),
		WithStream(stream),
	}

	return r.CreatePredictionWithOptions(ctx, opts...)
}

// ListPredictions returns a paginated list of predictions.
func (r *Client) ListPredictions(ctx context.Context) (*Page[Prediction], error) {
	response := &Page[Prediction]{}
	err := r.fetch(ctx, "GET", "/predictions", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	return response, nil
}

// GetPrediction retrieves a prediction from the Replicate API by its ID.
func (r *Client) GetPrediction(ctx context.Context, id string) (*Prediction, error) {
	prediction := &Prediction{}
	err := r.fetch(ctx, "GET", fmt.Sprintf("/predictions/%s", id), nil, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to get prediction: %w", err)
	}
	return prediction, nil
}

// CancelPrediction cancels a running prediction by its ID.
func (r *Client) CancelPrediction(ctx context.Context, id string) (*Prediction, error) {
	prediction := &Prediction{}
	err := r.fetch(ctx, "POST", fmt.Sprintf("/predictions/%s/cancel", id), nil, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel prediction: %w", err)
	}
	return prediction, nil
}
