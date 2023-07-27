package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var (
	ErrNoAuth = errors.New(`no auth token or token source provided -- perhaps you forgot to pass replicate.WithToken("...")`)

	defaultUserAgent = "replicate/go" // TODO: embed version information
	defaultBaseURL   = "https://api.replicate.com/v1"
)

// Client is a client for the Replicate API.
type Client struct {
	options *options
	c       *http.Client
}

type options struct {
	auth       string
	baseURL    string
	httpClient *http.Client
	userAgent  *string
}

// ClientOption is a function that modifies an options struct.
type ClientOption func(*options) error

// NewClient creates a new Replicate API client.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		options: &options{
			userAgent:  &defaultUserAgent,
			baseURL:    defaultBaseURL,
			httpClient: http.DefaultClient,
		},
	}

	var errs []error
	for _, option := range opts {
		err := option(c.options)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	if c.options.auth == "" {
		return nil, ErrNoAuth
	}

	c.c = c.options.httpClient

	return c, nil
}

// WithToken sets the auth token used by the client.
func WithToken(token string) ClientOption {
	return func(o *options) error {
		o.auth = token
		return nil
	}
}

// WithTokenFromEnv configures the client to use the auth token provided in the
// REPLICATE_API_TOKEN environment variable.
func WithTokenFromEnv() ClientOption {
	return func(o *options) error {
		token, ok := os.LookupEnv("REPLICATE_API_TOKEN")
		if !ok {
			return fmt.Errorf("REPLICATE_API_TOKEN environment variable not set")
		}
		if token == "" {
			return fmt.Errorf("REPLICATE_API_TOKEN environment variable is empty")
		}
		o.auth = token
		return nil
	}
}

// WithUserAgent sets the User-Agent header on requests made by the client.
func WithUserAgent(userAgent string) ClientOption {
	return func(o *options) error {
		o.userAgent = &userAgent
		return nil
	}
}

// WithBaseURL sets the base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(o *options) error {
		o.baseURL = baseURL
		return nil
	}
}

// WithHTTPClient sets the HTTP client used by the client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(o *options) error {
		o.httpClient = httpClient
		return nil
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

	url := constructURL(r.options.baseURL, path)
	request, err := http.NewRequestWithContext(ctx, method, url, bodyBuffer)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Token %s", r.options.auth))
	if r.options.userAgent != nil {
		request.Header.Set("User-Agent", *r.options.userAgent)
	}

	response, err := r.c.Do(request)
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
