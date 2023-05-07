package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

// Client represents a Replicate API client.
type Client struct {
	Auth       string
	UserAgent  *string
	BaseURL    string
	HTTPClient *http.Client
}

type Page[T any] struct {
	Previous *string `json:"previous,omitempty"`
	Next     *string `json:"next,omitempty"`
	Results  []T     `json:"results"`
}

// New creates a new Replicate API client.
func New(auth string, userAgent *string, baseURL *string) *Client {
	client := &http.Client{}

	if userAgent == nil {
		defaultUserAgent := "replicate-go"
		userAgent = &defaultUserAgent
	}

	if baseURL == nil {
		defaultBaseURL := "https://api.replicate.com/v1"
		baseURL = &defaultBaseURL
	}

	return &Client{
		Auth:       auth,
		UserAgent:  userAgent,
		BaseURL:    *baseURL,
		HTTPClient: client,
	}
}

// request makes an HTTP request to the Replicate API.
func (r *Client) request(ctx context.Context, method, path string, requestBody interface{}, responseTarget interface{}) error {
	// Initialize an empty buffer
	bodyBuffer := &bytes.Buffer{}

	// Marshal request body, if provided
	if requestBody != nil {
		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyBuffer = bytes.NewBuffer(bodyBytes)
	}

	// Construct request
	url := constructURL(r.BaseURL, path)
	request, err := http.NewRequestWithContext(ctx, method, url, bodyBuffer)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Token %s", r.Auth))
	if r.UserAgent != nil {
		request.Header.Set("User-Agent", *r.UserAgent)
	}

	// Send request
	response, err := r.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer response.Body.Close()

	// Read response body
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for API errors
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		var apiError APIError
		err := json.Unmarshal(responseBytes, &apiError)
		if err != nil {
			return fmt.Errorf("unable to parse API error: %v", err)
		}
		return apiError
	}

	// Unmarshal response into target, if provided
	if responseTarget != nil {
		if err := json.Unmarshal(responseBytes, &responseTarget); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func constructURL(baseUrl, route string) string {
	if strings.HasPrefix(route, "/") {
		route = route[1:]
	}
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl = baseUrl + "/"
	}
	return baseUrl + route
}

// Paginate takes a Page and the Client request method, and iterates through pages of results.
func Paginate[T any](ctx context.Context, client *Client, initialPage *Page[T]) (<-chan []T, <-chan error) {
	resultsChan := make(chan []T)
	errChan := make(chan error)

	go func() {
		defer close(resultsChan)
		defer close(errChan)

		resultsChan <- initialPage.Results
		nextURL := initialPage.Next

		for nextURL != nil {
			page := &Page[T]{}
			err := client.request(ctx, "GET", *nextURL, nil, page)
			if err != nil {
				errChan <- err
				return
			}

			resultsChan <- page.Results

			nextURL = page.Next
		}
	}()

	return resultsChan, errChan
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
