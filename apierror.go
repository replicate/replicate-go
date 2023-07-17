package replicate

import (
	"fmt"
)

// APIError represents an error returned by the Replicate API
type APIError struct {
	Detail string `json:"detail"`
}

// Error implements the error interface
func (e APIError) Error() string {
	return fmt.Sprintf("Replicate API error: %s", e.Detail)
}
