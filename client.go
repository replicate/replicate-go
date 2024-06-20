package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	envAuthToken = "REPLICATE_API_TOKEN"

	defaultUserAgent = "replicate/go" // TODO: embed version information
	defaultBaseURL   = "https://api.replicate.com/v1"

	defaultMaxRetries = 5
	defaultBackoff    = &ExponentialBackoff{
		Multiplier: 2,
		Base:       500 * time.Millisecond,
		Jitter:     50 * time.Millisecond,
	}

	ErrNoAuth       = errors.New(`no auth token or token source provided -- perhaps you forgot to pass replicate.WithToken("...")`)
	ErrEnvVarNotSet = fmt.Errorf("%s environment variable not set", envAuthToken)
	ErrEnvVarEmpty  = fmt.Errorf("%s environment variable is empty", envAuthToken)
)

// Client is a client for the Replicate API.
type Client struct {
	options *clientOptions
	c       *http.Client
}

type retryPolicy struct {
	maxRetries int
	backoff    Backoff
}

type clientOptions struct {
	auth        string
	baseURL     string
	httpClient  *http.Client
	retryPolicy *retryPolicy
	userAgent   *string
}

// ClientOption is a function that modifies an options struct.
type ClientOption func(*clientOptions) error

// NewClient creates a new Replicate API client.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		options: &clientOptions{
			userAgent: &defaultUserAgent,
			baseURL:   defaultBaseURL,
			retryPolicy: &retryPolicy{
				maxRetries: defaultMaxRetries,
				backoff:    defaultBackoff,
			},
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
		err := errors.Join(errs...)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to apply options")
	}

	if c.options.auth == "" {
		return nil, ErrNoAuth
	}

	c.c = c.options.httpClient

	return c, nil
}

// WithToken sets the auth token used by the client.
func WithToken(token string) ClientOption {
	return func(o *clientOptions) error {
		o.auth = token
		return nil
	}
}

// WithTokenFromEnv configures the client to use the auth token provided in the
// REPLICATE_API_TOKEN environment variable.
func WithTokenFromEnv() ClientOption {
	return func(o *clientOptions) error {
		token, ok := os.LookupEnv(envAuthToken)
		if !ok {
			return ErrEnvVarNotSet
		}
		if token == "" {
			return ErrEnvVarEmpty
		}
		o.auth = token
		return nil
	}
}

// WithUserAgent sets the User-Agent header on requests made by the client.
func WithUserAgent(userAgent string) ClientOption {
	return func(o *clientOptions) error {
		o.userAgent = &userAgent
		return nil
	}
}

// WithBaseURL sets the base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(o *clientOptions) error {
		o.baseURL = baseURL
		return nil
	}
}

// WithHTTPClient sets the HTTP client used by the client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(o *clientOptions) error {
		o.httpClient = httpClient
		return nil
	}
}

// WithRetryPolicy sets the retry policy used by the client.
func WithRetryPolicy(maxRetries int, backoff Backoff) ClientOption {
	return func(o *clientOptions) error {
		o.retryPolicy = &retryPolicy{
			maxRetries: maxRetries,
			backoff:    backoff,
		}
		return nil
	}
}

func (r *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := constructURL(r.options.baseURL, path)
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.options.auth))
	if r.options.userAgent != nil {
		request.Header.Set("User-Agent", *r.options.userAgent)
	}

	return request, nil
}

func (r *Client) do(request *http.Request, out interface{}) error {
	maxRetries := r.options.retryPolicy.maxRetries
	backoff := r.options.retryPolicy.backoff

	var apiError *APIError
	attempts := 0
	for ok := true; ok; ok = attempts < maxRetries {
		response, err := r.c.Do(request)
		if err != nil || response == nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer response.Body.Close()

		responseBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if response.StatusCode < 200 || response.StatusCode >= 400 {
			apiError = unmarshalAPIError(response, responseBytes)
			if !r.shouldRetry(response, request.Method) {
				return apiError
			}

			delay := backoff.NextDelay(attempts)

			retryAfter := response.Header.Get("Retry-After")
			if retryAfter != "" {
				if parsedDelay, parseErr := time.Parse(time.RFC1123, retryAfter); parseErr == nil {
					delay = time.Until(parsedDelay)
				} else if seconds, convErr := strconv.Atoi(retryAfter); convErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}

			if delay > 0 {
				time.Sleep(delay)
			}

			attempts++
		} else {
			if out != nil {
				if err := json.Unmarshal(responseBytes, &out); err != nil {
					return fmt.Errorf("failed to unmarshal response: %w", err)
				}
			}

			return nil
		}
	}

	if apiError != nil {
		return apiError
	}

	if attempts > 0 {
		return fmt.Errorf("request failed after %d attempts", maxRetries)
	}

	return fmt.Errorf("request failed")
}

// fetch makes an HTTP request to Replicate's API.
func (r *Client) fetch(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	bodyBuffer := &bytes.Buffer{}
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyBuffer = bytes.NewBuffer(bodyBytes)
	}

	request, err := r.newRequest(ctx, method, path, bodyBuffer)
	if err != nil {
		return err
	}

	return r.do(request, out)
}

// shouldRetry returns true if the request should be retried.
//
// - GET requests should be retried if the response status code is 429 or 5xx.
// - Other requests should be retried if the response status code is 429.
func (r *Client) shouldRetry(response *http.Response, method string) bool {
	if method == http.MethodGet {
		return response.StatusCode == 429 || (response.StatusCode >= 500 && response.StatusCode < 600)
	}

	return response.StatusCode == 429
}

func constructURL(baseURL, route string) string {
	route = strings.TrimPrefix(route, "/")

	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return baseURL + route
}
