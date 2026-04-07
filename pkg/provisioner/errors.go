// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: Apache-2.0

package provisioner

import (
	"errors"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// ClassifyError maps a GitLab API error to a formae OperationErrorCode.
func ClassifyError(err error) resource.OperationErrorCode {
	var glErr *gitlab.ErrorResponse
	if errors.As(err, &glErr) {
		return classifyHTTPStatus(glErr.Response.StatusCode)
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

// IsNotFound returns true if the error is a GitLab 404 response.
func IsNotFound(err error) bool {
	var glErr *gitlab.ErrorResponse
	if errors.As(err, &glErr) {
		return glErr.Response.StatusCode == http.StatusNotFound
	}
	return false
}
