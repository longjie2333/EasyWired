package backend

import (
	"errors"
	"fmt"
)

var (
	ErrBackendNotSupported   = errors.New("backend not supported")
	ErrBackendNotImplemented = errors.New("backend not implemented")
)

type BackendError struct {
	Code    string
	Message string
	Err     error
}

func (e *BackendError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *BackendError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func notImplemented(name string) error {
	return &BackendError{
		Code:    "BACKEND_NOT_IMPLEMENTED",
		Message: fmt.Sprintf("%s backend is reserved but not implemented; use wgconfig-file", name),
		Err:     ErrBackendNotImplemented,
	}
}
