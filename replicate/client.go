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

// Client is a client for the Replicate API.
type Client struct {
	Auth       string
	UserAgent  *string
	BaseURL    string
	HTTPClient *http.Client
}

// ClientOption is a function that modifies a Client.
type ClientOption func(*Client)

// NewClient creates a new Replicate API client.
func NewClient(auth string, options ...ClientOption) *Client {
	client := &http.Client{}
	defaultUserAgent := "replicate-go"
	defaultBaseURL := "https://api.replicate.com/v1"

	c := &Client{
		Auth:       auth,
		UserAgent:  &defaultUserAgent,
		BaseURL:    defaultBaseURL,
		HTTPClient: client,
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// WithUserAgent sets the User-Agent header on requests made by the client.
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.UserAgent = &userAgent
	}
}

// WithBaseURL sets the base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.BaseURL = baseURL
	}
}

// WithHTTPClient sets the HTTP client used by the client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.HTTPClient = httpClient
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
