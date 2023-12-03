package replicate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents an error returned by the Replicate API
type APIError struct {
	Type     string `json:"type,omitempty"`
	Status   int    `json:"status,omitempty"`
	Title    string `json:"title,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func unmarshalAPIError(resp *http.Response, data []byte) *APIError {
	apiError := APIError{}
	err := json.Unmarshal(data, &apiError)
	if err != nil {
		apiError.Detail = fmt.Sprintf("Unknown error: %s", err)
	}

	if apiError.Status == 0 && resp != nil {
		apiError.Status = resp.StatusCode
	}

	return &apiError
}

// Error implements the error interface
func (e APIError) Error() string {
	components := []string{}
	if e.Type != "" {
		components = append(components, e.Type)
	}

	if e.Title != "" {
		components = append(components, e.Title)
	}

	if e.Detail != "" {
		components = append(components, e.Detail)
	}

	output := strings.Join(components, ": ")
	if output == "" {
		output = "Unknown error"
	}

	if e.Instance != "" {
		output = fmt.Sprintf("%s (%s)", output, e.Instance)
	}

	return output
}

func (e *APIError) WriteHTTPResponse(w http.ResponseWriter) {
	status := http.StatusBadGateway
	if e.Status != 0 {
		status = e.Status
	}

	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
