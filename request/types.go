package request

import (
	"fmt"

	ctrlserver "EasyWired/server"
)

type APIError struct {
	StatusCode int
	Status     string
	Message    string
	Details    []ctrlserver.ErrorDetail
}

func (e *APIError) Error() string {
	if e == nil {
		return "request failed"
	}

	if e.Message == "" {
		return fmt.Sprintf("request failed: %s", e.Status)
	}

	return fmt.Sprintf("request failed (%d): %s", e.StatusCode, e.Message)
}
