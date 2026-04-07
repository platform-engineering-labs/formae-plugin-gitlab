// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package provisioner

import (
	"errors"
	"fmt"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// StatusError wraps an error with an HTTP status code from the GitLab response.
// The GitLab Go SDK inconsistently returns plain errors (not *gitlab.ErrorResponse)
// for many status codes — particularly 404s. This type preserves the status
// so ClassifyError and IsNotFound work uniformly.
type StatusError struct {
	Err        error
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Err.Error())
}

func (e *StatusError) Unwrap() error {
	return e.Err
}

// WrapError enriches a GitLab SDK error with the HTTP response status code.
// Call this immediately after any GitLab SDK call to normalize error handling:
//
//	result, resp, err := client.Something.Do(...)
//	if err = provisioner.WrapError(err, resp); err != nil { ... }
//
// If the error is already a *gitlab.ErrorResponse, it's returned as-is.
// If it's a plain error but the response has a status code, it's wrapped
// in a *StatusError so ClassifyError and IsNotFound can read the status.
func WrapError(err error, resp *gitlab.Response) error {
	if err == nil {
		return nil
	}
	var glErr *gitlab.ErrorResponse
	if errors.As(err, &glErr) {
		return err
	}
	if resp != nil {
		return &StatusError{Err: err, StatusCode: resp.StatusCode}
	}
	return err
}

// ClassifyError maps a GitLab API error to a formae OperationErrorCode.
// Handles both *gitlab.ErrorResponse and *StatusError (from WrapError).
func ClassifyError(err error) resource.OperationErrorCode {
	var glErr *gitlab.ErrorResponse
	if errors.As(err, &glErr) {
		return classifyHTTPStatus(glErr.Response.StatusCode)
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return classifyHTTPStatus(statusErr.StatusCode)
	}
	return resource.OperationErrorCodeInternalFailure
}

func classifyHTTPStatus(status int) resource.OperationErrorCode {
	switch status {
	case http.StatusBadRequest:
		return resource.OperationErrorCodeInvalidRequest
	case http.StatusUnauthorized:
		return resource.OperationErrorCodeInvalidCredentials
	case http.StatusForbidden:
		return resource.OperationErrorCodeAccessDenied
	case http.StatusNotFound:
		return resource.OperationErrorCodeNotFound
	case http.StatusConflict:
		return resource.OperationErrorCodeAlreadyExists
	case http.StatusUnprocessableEntity:
		return resource.OperationErrorCodeInvalidRequest
	case http.StatusTooManyRequests:
		return resource.OperationErrorCodeThrottling
	default:
		if status >= 500 {
			return resource.OperationErrorCodeServiceInternalError
		}
		return resource.OperationErrorCodeInternalFailure
	}
}

// IsNotFound returns true if the error indicates a 404.
// Handles both *gitlab.ErrorResponse and *StatusError.
func IsNotFound(err error) bool {
	return ClassifyError(err) == resource.OperationErrorCodeNotFound
}
