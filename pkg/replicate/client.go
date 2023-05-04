package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Client struct {
	Auth       string
	UserAgent  *string
	BaseURL    string
	HTTPClient *http.Client
}

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
	request.Header.Set("User-Agent", *r.UserAgent)

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
		var apiError struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(responseBytes, &apiError); err != nil {
			return fmt.Errorf("failed to unmarshal error message: %w", err)
		}
		return errors.New(apiError.Message)
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
