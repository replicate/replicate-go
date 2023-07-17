package replicate

import (
	"fmt"
	"net/http"
	"os"
)

type options struct {
	auth       string
	baseURL    string
	httpClient *http.Client
	userAgent  *string
}

// Option is a function that modifies an options struct.
type Option func(*options) error

// WithToken sets the auth token used by the client.
func WithToken(token string) Option {
	return func(o *options) error {
		o.auth = token
		return nil
	}
}

// WithTokenFromEnv configures the client to use the auth token provided in the
// REPLICATE_API_TOKEN environment variable.
func WithTokenFromEnv() Option {
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
func WithUserAgent(userAgent string) Option {
	return func(o *options) error {
		o.userAgent = &userAgent
		return nil
	}
}

// WithBaseURL sets the base URL for the client.
func WithBaseURL(baseURL string) Option {
	return func(o *options) error {
		o.baseURL = baseURL
		return nil
	}
}

// WithHTTPClient sets the HTTP client used by the client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(o *options) error {
		o.httpClient = httpClient
		return nil
	}
}
