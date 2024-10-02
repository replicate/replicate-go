package replicate

import (
	"bytes"
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
	useFileOutput  bool
	blockUntilDone bool
}

// FileOutput is a custom type that implements io.ReadCloser and includes a URL field
type FileOutput struct {
	io.ReadCloser
	URL string
}

// WithFileOutput configures the run to automatically convert URLs in output to FileOutput objects
func WithFileOutput() RunOption {
	return func(o *runOptions) {
		o.useFileOutput = true
	}
}

// WithBlockUntilDone configures the run to block until the prediction is done
func WithBlockUntilDone() RunOption {
	return func(o *runOptions) {
		o.blockUntilDone = true
	}
}

// RunWithOptions runs a model with specified options
func (r *Client) RunWithOptions(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook, opts ...RunOption) (PredictionOutput, error) {
	// Initialize options
	options := runOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	// Parse the identifier to extract version
	id, err := ParseIdentifier(identifier)
	if err != nil {
		return nil, err
	}

	// Prepare the data for the prediction request
	data := map[string]interface{}{}
	path := "/predictions"

	// Set the model path or version in the data
	if id.Version == nil {
		path = fmt.Sprintf("/models/%s/%s/predictions", id.Owner, id.Name)
	} else {
		data["version"] = *id.Version
	}

	// Create the prediction request
	req, err := r.createPredictionRequest(ctx, path, data, input, webhook, false)
	if err != nil {
		return nil, err
	}

	// Set the Prefer header if blockUntilDone is true
	if options.blockUntilDone {
		req.Header.Set("Prefer", "wait")
	}

	// Execute the request and obtain the prediction
	prediction := &Prediction{}
	if err := r.do(req, prediction); err != nil {
		return nil, err
	}

	// Check if the prediction is done based on blocking preference and status
	isDone := options.blockUntilDone && prediction.Status != Starting
	if !isDone {
		// Wait for the prediction to complete
		err = r.Wait(ctx, prediction)
		if err != nil {
			return nil, err
		}
	}

	// Check for model error in the prediction
	if prediction.Error != nil {
		return nil, &ModelError{Prediction: prediction}
	}

	// Transform the output based on the options
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

func readDataURI(uri string) (*FileOutput, error) {
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
	var reader io.Reader
	if strings.HasSuffix(mediatype, ";base64") {
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		reader = strings.NewReader(data)
	}
	return &FileOutput{
		ReadCloser: io.NopCloser(reader),
		URL:        uri,
	}, nil
}

func readHTTP(ctx context.Context, url string, client *Client) (*FileOutput, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("HTTP request failed to get a response")
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}

	return &FileOutput{
		ReadCloser: resp.Body,
		URL:        url,
	}, nil
}
