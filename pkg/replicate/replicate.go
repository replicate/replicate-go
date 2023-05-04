package replicate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)


type Page struct {
	Previous *string     `json:"previous,omitempty"`
	Next     *string     `json:"next,omitempty"`
	Results  interface{} `json:"results"`
}

type Collection struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Models      []Model `json:"models"`
}

type Model struct {
	URL            string        `json:"url"`
	Owner          string        `json:"owner"`
	Name           string        `json:"name"`
	Description    *string       `json:"description,omitempty"`
	Visibility     string        `json:"visibility"`
	GithubURL      *string       `json:"github_url,omitempty"`
	PaperURL       *string       `json:"paper_url,omitempty"`
	LicenseURL     *string       `json:"license_url,omitempty"`
	RunCount       int           `json:"run_count"`
	CoverImageURL  *string       `json:"cover_image_url,omitempty"`
	DefaultExample *Prediction   `json:"default_example,omitempty"`
	LatestVersion  *ModelVersion `json:"latest_version,omitempty"`
}

type ModelVersion struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	CogVersion    string      `json:"cog_version"`
	OpenAPISchema interface{} `json:"openapi_schema"`
}

type Source string

const (
	Web Source = "web"
	API Source = "api"
)

type Prediction struct {
	ID                  string             `json:"id"`
	Status              Status             `json:"status"`
	Version             string             `json:"version"`
	Input               PredictionInput    `json:"input"`
	Output              PredictionOutput   `json:"output,omitempty"`
	Source              Source             `json:"source"`
	Error               interface{}        `json:"error,omitempty"`
	Logs                *string            `json:"logs,omitempty"`
	Metrics             *Metrics           `json:"metrics,omitempty"`
	Webhook             *string            `json:"webhook,omitempty"`
	WebhookEventsFilter []WebhookEventType `json:"webhook_events_filter,omitempty"`
	CreatedAt           string             `json:"created_at"`
	UpdatedAt           string             `json:"updated_at"`
	CompletedAt         *string            `json:"completed_at,omitempty"`
}

type PredictionInput map[string]interface{}
type PredictionOutput interface{}

type Metrics struct {
	PredictTime *float64 `json:"predict_time,omitempty"`
}

type Training Prediction

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

func (r *Client) GetCollection(ctx context.Context, slug string) (*Collection, error) {
	collection := &Collection{}
	err := r.request(ctx, "GET", fmt.Sprintf("/collections/%s", slug), nil, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return collection, nil
}

func (r *Client) Run(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (PredictionOutput, error) {
	namePattern := `[a-zA-Z0-9]+(?:(?:[._]|__|[-]*)[a-zA-Z0-9]+)*`
	pattern := fmt.Sprintf(`^(?P<owner>%s)/(?P<name>%s):(?P<version>[0-9a-fA-F]+)$`, namePattern, namePattern)

	regex := regexp.MustCompile(pattern)
	match := regex.FindStringSubmatch(identifier)

	if len(match) == 0 {
		return nil, errors.New("invalid version. it must be in the format \"owner/name:version\"")
	}

	version := ""
	for i, name := range regex.SubexpNames() {
		if name == "version" {
			version = match[i]
		}
	}

	prediction, err := r.CreatePrediction(ctx, version, input, webhook)
	if err != nil {
		return nil, err
	}

	return prediction.Output, err

}

// 	if err != nil {
// 		return nil, err
// 	}

// 	if options.Wait != nil {
// 		prediction, err = r.Wait(prediction, *options.Wait)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	if prediction.Status == "failed" {
// 		return nil, fmt.Errorf("prediction failed: %v", prediction.Error)
// 	}

// 	return prediction.Output, nil
// }

// func (r *Replicate) CreatePrediction(options PredictionOptions) (*Prediction, error) {
// 	// implement logic to create a new prediction
// }

// func (r *Replicate) Wait(prediction *Prediction, options WaitOptions) (*Prediction, error) {
// 	// implement logic to wait for a prediction to finish
// }

// type PredictionOptions struct {
// 	Version             string
// 	Input               map[string]interface{}
// 	Webhook             *string
// 	WebhookEventsFilter []WebhookEventType
// }

// type WaitOptions struct {
// 	Interval    *int
// 	MaxAttempts *int
// }
