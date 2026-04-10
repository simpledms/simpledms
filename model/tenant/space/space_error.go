package space

import "errors"

var (
	ErrUserAlreadyAssignedToSpace    = errors.New("user already assigned to this space")
	ErrCannotUnassignYourselfInSpace = errors.New("cannot unassign yourself from a space")
)
