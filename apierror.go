package replicate

import (
	"encoding/json"
	"fmt"
)

// APIError represents an error returned by the Replicate API
type APIError struct {
	Detail string `json:"detail"`
}

func unmarshalAPIError(data []byte) *APIError {
	apiError := &APIError{}
	err := json.Unmarshal(data, apiError)
	if err != nil {
		apiError.Detail = fmt.Sprintf("Unknown error: %s", err)
	}

	return apiError
}

// Error implements the error interface
func (e APIError) Error() string {
	return fmt.Sprintf("Replicate API error: %s", e.Detail)
}
