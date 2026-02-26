package handler

import (
	"errors"
	"strings"
	"time"
)

const (
	MaxNameLength     = 200
	MaxFullNameLength = 200
	MaxPositionLength = 200
	DateFormat        = "2006-01-02"
)

var (
	ErrNameRequired         = errors.New("name is required")
	ErrNameTooLong          = errors.New("name must not exceed 200 characters")
	ErrNameCannotBeEmpty    = errors.New("name cannot be empty")
	ErrFullNameRequired     = errors.New("full_name is required")
	ErrFullNameTooLong      = errors.New("full_name must not exceed 200 characters")
	ErrPositionRequired     = errors.New("position is required")
	ErrPositionTooLong      = errors.New("position must not exceed 200 characters")
	ErrHiredAtEmpty         = errors.New("hired_at cannot be empty")
	ErrHiredAtInvalidFormat = errors.New("hired_at must be a valid date in format YYYY-MM-DD")
	ErrModeInvalid          = errors.New("mode must be either 'cascade' or 'reassign'")
	ErrReassignIDRequired   = errors.New("reassign_to_department_id is required when mode=reassign")
)

type CreateDepartmentRequest struct {
	ParentID *uint  `json:"parent_id"`
	Name     string `json:"name"`
}

func (r *CreateDepartmentRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	return nil
}

type CreateEmployeeRequest struct {
	HiredAt  *string `json:"hired_at"`
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
}

func (r *CreateEmployeeRequest) Validate() error {
	r.FullName = strings.TrimSpace(r.FullName)
	r.Position = strings.TrimSpace(r.Position)

	if r.FullName == "" {
		return ErrFullNameRequired
	}
	if len(r.FullName) > MaxFullNameLength {
		return ErrFullNameTooLong
	}
	if r.Position == "" {
		return ErrPositionRequired
	}
	if len(r.Position) > MaxPositionLength {
		return ErrPositionTooLong
	}

	if r.HiredAt != nil {
		trimmed := strings.TrimSpace(*r.HiredAt)
		if trimmed == "" {
			return ErrHiredAtEmpty
		}
		parsed, err := time.Parse(DateFormat, trimmed)
		if err != nil {
			return ErrHiredAtInvalidFormat
		}
		*r.HiredAt = parsed.Format(DateFormat) // нормализуем формат
	}
	return nil
}

type UpdateDepartmentRequest struct {
	Name     *string `json:"name"`
	ParentID *uint   `json:"parent_id"`
}

func (r *UpdateDepartmentRequest) Validate() error {
	if r.Name != nil {
		trimmed := strings.TrimSpace(*r.Name)
		if trimmed == "" {
			return ErrNameCannotBeEmpty
		}
		if len(trimmed) > MaxNameLength {
			return ErrNameTooLong
		}
		*r.Name = trimmed
	}
	return nil
}

// DeleteDepartmentRequest используется для парсинга query-параметров (gorilla/schema).
type DeleteDepartmentRequest struct {
	ReassignToDepartmentID *uint  `schema:"reassign_to_department_id"`
	Mode                   string `schema:"mode"`
}

func (r *DeleteDepartmentRequest) Validate() error {
	r.Mode = strings.TrimSpace(strings.ToLower(r.Mode))
	if r.Mode != "cascade" && r.Mode != "reassign" {
		return ErrModeInvalid
	}
	if r.Mode == "reassign" && r.ReassignToDepartmentID == nil {
		return ErrReassignIDRequired
	}
	return nil
}
