package service

import "errors"

var (
	ErrDepartmentNotFound         = errors.New("department not found")
	ErrDuplicateName              = errors.New("department with this name already exists under the same parent")
	ErrCycleDetected              = errors.New("cannot move department into its own descendant")
	ErrReassignDepartmentNotFound = errors.New("reassign target department not found")
)
