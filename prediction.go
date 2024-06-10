package replicate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type Source string

const (
	SourceWeb Source = "web"
	SourceAPI Source = "api"
)

type Prediction struct {
	ID                  string             `json:"id"`
	Status              Status             `json:"status"`
	Model               string             `json:"model"`
	Version             string             `json:"version"`
	Input               PredictionInput    `json:"input"`
	Output              PredictionOutput   `json:"output,omitempty"`
	Source              Source             `json:"source"`
	Error               interface{}        `json:"error,omitempty"`
	Logs                *string            `json:"logs,omitempty"`
	Metrics             *PredictionMetrics `json:"metrics,omitempty"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
	URLs                map[string]string  `json:"urls,omitempty"`
	CreatedAt           string             `json:"created_at"`
	StartedAt           *string            `json:"started_at,omitempty"`
	CompletedAt         *string            `json:"completed_at,omitempty"`

	rawJSON json.RawMessage `json:"-"`
}

func (p *Prediction) RawJSON() json.RawMessage {
	return p.rawJSON
}

var _ json.Unmarshaler = (*Prediction)(nil)

func (p *Prediction) UnmarshalJSON(data []byte) error {
	p.rawJSON = data
	type Alias Prediction
	alias := &struct{ *Alias }{Alias: (*Alias)(p)}
	return json.Unmarshal(data, alias)
}

type PredictionInput map[string]interface{}
type PredictionOutput interface{}

type PredictionMetrics struct {
	PredictTime      *float64 `json:"predict_time,omitempty"`
	TotalTime        *float64 `json:"total_time,omitempty"`
	InputTokenCount  *int     `json:"input_token_count,omitempty"`
	OutputTokenCount *int     `json:"output_token_count,omitempty"`
	TimeToFirstToken *float64 `json:"time_to_first_token,omitempty"`
	TokensPerSecond  *float64 `json:"tokens_per_second,omitempty"`
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

// CreatePrediction sends a request to the Replicate API to create a prediction.
func (r *Client) CreatePrediction(ctx context.Context, version string, input PredictionInput, webhook *Webhook, stream bool) (*Prediction, error) {
	// Convert File objects in input to their "get" URL value
	for key, value := range input {
		if file, ok := value.(*File); ok {
			input[key] = file.URLs["get"]
		}
	}

	data := map[string]interface{}{
		"version": version,
		"input":   input,
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
	err := r.fetch(ctx, http.MethodPost, "/predictions", data, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to create prediction: %w", err)
	}

	return prediction, nil
}

// ListPredictions returns a paginated list of predictions.
func (r *Client) ListPredictions(ctx context.Context) (*Page[Prediction], error) {
	response := &Page[Prediction]{}
	err := r.fetch(ctx, http.MethodGet, "/predictions", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictions: %w", err)
	}
	return response, nil
}

// GetPrediction retrieves a prediction from the Replicate API by its ID.
func (r *Client) GetPrediction(ctx context.Context, id string) (*Prediction, error) {
	prediction := &Prediction{}
	err := r.fetch(ctx, http.MethodGet, fmt.Sprintf("/predictions/%s", id), nil, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to get prediction: %w", err)
	}
	return prediction, nil
}

// CancelPrediction cancels a running prediction by its ID.
func (r *Client) CancelPrediction(ctx context.Context, id string) (*Prediction, error) {
	prediction := &Prediction{}
	err := r.fetch(ctx, http.MethodPost, fmt.Sprintf("/predictions/%s/cancel", id), nil, prediction)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel prediction: %w", err)
	}
	return prediction, nil
}
