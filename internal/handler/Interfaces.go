package handler

import (
	"time"

	"org-api/internal/domain"
)

// DepartmentService — интерфейс сервиса подразделений (используется в хендлере и тестах).
type DepartmentService interface {
	Create(name string, parentID *uint) (*domain.Department, error)
	Update(id uint, name *string, parentID *uint) (*domain.Department, error)
	GetWithTree(
		id uint,
		depth int,
		includeEmployees bool,
		sortBy string,
	) (*domain.Department, error)
	Delete(id uint, mode string, reassignTo *uint) error
}

// EmployeeService — интерфейс сервиса сотрудников.
type EmployeeService interface {
	Create(
		departmentID uint,
		fullName, position string,
		hiredAt *time.Time,
	) (*domain.Employee, error)
}
