// Package audit provides error types
package audit

import "errors"

var (
	ErrAuditLogNotFound = errors.New("audit log not found")
)
