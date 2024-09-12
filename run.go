package replicate

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RunOption is a function that modifies RunOptions
type RunOption func(*runOptions)

// runOptions represents options for running a model
type runOptions struct {
	useFileOutput bool
}

// WithFileOutput sets the UseFileOutput option to true
func WithFileOutput() RunOption {
	return func(o *runOptions) {
		o.useFileOutput = true
	}
}

// RunWithOptions runs a model with specified options
func (r *Client) RunWithOptions(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook, opts ...RunOption) (PredictionOutput, error) {
	options := runOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	id, err := ParseIdentifier(identifier)
	if err != nil {
		return nil, err
	}

	if id.Version == nil {
		return nil, errors.New("version must be specified")
	}

	prediction, err := r.CreatePrediction(ctx, *id.Version, input, webhook, false)
	if err != nil {
		return nil, err
	}

	err = r.Wait(ctx, prediction)
	if err != nil {
		return nil, err
	}

	if prediction.Error != nil {
		return nil, &ModelError{Prediction: prediction}
	}

	if options.useFileOutput {
		return transformOutput(ctx, prediction.Output, r)
	}

	return prediction.Output, nil
}

// Run runs a model and returns the output
func (r *Client) Run(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (PredictionOutput, error) {
	return r.RunWithOptions(ctx, identifier, input, webhook)
}

func transformOutput(ctx context.Context, value interface{}, client *Client) (interface{}, error) {
	var err error
	switch v := value.(type) {
	case map[string]interface{}:
		for k, val := range v {
			v[k], err = transformOutput(ctx, val, client)
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case []interface{}:
		for i, val := range v {
			v[i], err = transformOutput(ctx, val, client)
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case string:
		if strings.HasPrefix(v, "data:") {
			return readDataURI(v)
		}
		if strings.HasPrefix(v, "https:") || strings.HasPrefix(v, "http:") {
			return readHTTP(ctx, v, client)
		}
		return v, nil
	}
	return value, nil
}

func readDataURI(uri string) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "data" {
		return nil, errors.New("not a data URI")
	}
	mediatype, data, found := strings.Cut(u.Opaque, ",")
	if !found {
		return nil, errors.New("invalid data URI format")
	}
	if strings.HasSuffix(mediatype, ";base64") {
		return base64.StdEncoding.DecodeString(data)
	}
	return []byte(data), nil
}

func readHTTP(ctx context.Context, url string, client *Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.c.Do(req)
	if resp == nil || resp.Body == nil {
		return nil, errors.New("HTTP request failed to get a response")
	}
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
