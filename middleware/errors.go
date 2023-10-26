package middleware

import "errors"

type APIError error

var (
	// General
	APIErrorUnknown        APIError = errors.New("unknownError")
	APIErrorInvalidRequest APIError = errors.New("invalidRequest")
	APIErrorNotFound       APIError = errors.New("notFound")

	// Auth
	APIErrorUnauthorizedDevice APIError = errors.New("unauthorizedDevice")
	APIErrorBannedDevice       APIError = errors.New("bannedDevice")

	// Other
	APIErrorDeviceNotEnrolled APIError = errors.New("deviceNotEnrolled")
	APIErrorServerInactive    APIError = errors.New("serverInactive")
	APIErrorServerNotCovered  APIError = errors.New("serverNotCovered")
)
