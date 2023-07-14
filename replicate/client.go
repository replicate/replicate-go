package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Client represents a Replicate API client.
type Client struct {
	Auth       string
	UserAgent  *string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Replicate API client.
func NewClient(auth string) *Client {
	return NewClientWithOptions(auth, nil, nil)
}

// NewClientWithOptions creates a new Replicate API client with a custom user agent and base URL.
func NewClientWithOptions(auth string, userAgent *string, baseURL *string) *Client {
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
func (r *Client) request(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	bodyBuffer := &bytes.Buffer{}
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyBuffer = bytes.NewBuffer(bodyBytes)
	}

	url := constructURL(r.BaseURL, path)
	request, err := http.NewRequestWithContext(ctx, method, url, bodyBuffer)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Token %s", r.Auth))
	if r.UserAgent != nil {
		request.Header.Set("User-Agent", *r.UserAgent)
	}

	response, err := r.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		var apiError APIError
		err := json.Unmarshal(responseBytes, &apiError)
		if err != nil {
			return fmt.Errorf("unable to parse API error: %v", err)
		}
		return apiError
	}

	if out != nil {
		if err := json.Unmarshal(responseBytes, &out); err != nil {
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
