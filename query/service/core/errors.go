package core

import "errors"

// Standard errors for query service
var (
	ErrLogNotFound      = errors.New("log not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrBlockchainError  = errors.New("blockchain query failed")
)
