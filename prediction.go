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

type createPredictionBody struct {
	Version             *string            `json:"version,omitempty"`
	Input               PredictionInput    `json:"input"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
	Stream              *bool              `json:"stream,omitempty"`
}

type CreatePredictionOption interface {
	applyToCreatePredictionPath(*string)
	applyToCreatePredictionBody(*createPredictionBody)
}

var _ CreatePredictionOption = withModelOption{}
var _ CreatePredictionOption = withModelVersionOption{}
var _ CreatePredictionOption = withVersionOption{}
var _ CreatePredictionOption = withDeploymentOption{}
var _ CreatePredictionOption = withInputOption{}
var _ CreatePredictionOption = withWebhookOption{}
var _ CreatePredictionOption = withStreamOption{}

//

type withModelOption struct {
	owner string
	name  string
}

func (o withModelOption) applyToCreatePredictionPath(path *string) {
	(*path) = fmt.Sprintf("/models/%s/%s/predictions", o.owner, o.name)
}

func (o withModelOption) applyToCreatePredictionBody(body *createPredictionBody) {
	body.Version = nil
}

func WithModel(owner string, name string) CreatePredictionOption {
	return withModelOption{owner: owner, name: name}
}

//

type withModelVersionOption struct {
	owner   string
	name    string
	version string
}

func (o withModelVersionOption) applyToCreatePredictionPath(path *string) {
	(*path) = fmt.Sprintf("/models/%s/%s/versions/%s/predictions", o.owner, o.name, o.version)
}

func (o withModelVersionOption) applyToCreatePredictionBody(body *createPredictionBody) {
	body.Version = nil
}

func WithModelVersion(_owner string, _name string, version string) withModelVersionOption {
	return withModelVersionOption{owner: _owner, name: _name, version: version}
}

//

type withVersionOption struct {
	version string
}

func (o withVersionOption) applyToCreatePredictionPath(path *string) {
	(*path) = "/predictions"
}

func (o withVersionOption) applyToCreatePredictionBody(body *createPredictionBody) {
	body.Version = &o.version
}

func WithVersion(version string) withVersionOption {
	return withVersionOption{version: version}
}

//

type withDeploymentOption struct {
	owner string
	name  string
}

func (o withDeploymentOption) applyToCreatePredictionPath(path *string) {
	(*path) = fmt.Sprintf("/deployments/%s/%s/predictions", o.owner, o.name)
}

func (o withDeploymentOption) applyToCreatePredictionBody(body *createPredictionBody) {
	body.Version = nil
}

func WithDeployment(owner string, name string) withDeploymentOption {
	return withDeploymentOption{owner: owner, name: name}
}

//

type withInputOption struct {
	input map[string]interface{}
}

func (o withInputOption) applyToCreatePredictionPath(path *string) {}
func (o withInputOption) applyToCreatePredictionBody(body *createPredictionBody) {
	body.Input = o.input
}

func WithInput(input map[string]interface{}) withInputOption {
	return withInputOption{input: input}
}

//

type withWebhookOption struct {
	webhook *Webhook
}

func (o withWebhookOption) applyToCreatePredictionPath(path *string) {}
func (o withWebhookOption) applyToCreatePredictionBody(body *createPredictionBody) {
	if o.webhook != nil {
		body.Webhook = &o.webhook.URL
		if len(o.webhook.Events) > 0 {
			body.WebhookEventsFilter = o.webhook.Events
		}
	}
}

func WithWebhook(webhook *Webhook) withWebhookOption {
	return withWebhookOption{webhook: webhook}
}

type withStreamOption struct {
	stream bool
}

func (o withStreamOption) applyToCreatePredictionPath(path *string) {}
func (o withStreamOption) applyToCreatePredictionBody(body *createPredictionBody) {
	if o.stream {
		body.Stream = &o.stream
	}
}

func WithStream(stream bool) withStreamOption {
	return withStreamOption{stream: stream}
}

func (r *Client) CreatePredictionWithOptions(ctx context.Context, opts ...CreatePredictionOption) (*Prediction, error) {
	path := ""
	body := &createPredictionBody{}
	for _, opt := range opts {
		opt.applyToCreatePredictionPath(&path)
		opt.applyToCreatePredictionBody(body)
	}

	prediction := &Prediction{}
	err := r.fetch(ctx, "POST", path, body, prediction)
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
