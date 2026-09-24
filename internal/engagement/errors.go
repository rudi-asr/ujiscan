// Package engagement provides error types
package engagement

import "errors"

var (
	ErrEngagementNotFound       = errors.New("engagement not found")
	ErrFindingNotFound          = errors.New("finding not found")
	ErrInvalidStatus            = errors.New("invalid status transition")
	ErrAlreadyAssigned          = errors.New("user already assigned to engagement")
	ErrAlreadyMarkedFalsePositive = errors.New("finding already marked as false positive")
	ErrCannotApproveFalsePositive = errors.New("cannot approve false positive finding")
	ErrInvalidFindingStatus     = errors.New("invalid finding status for this operation")
)
