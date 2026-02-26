package service

import (
	"log/slog"
	"time"

	"org-api/internal/domain"
	"org-api/internal/repository"
)

type EmployeeService struct {
	empRepo  repository.EmployeeRepository
	deptRepo repository.DepartmentRepository
	logger   *slog.Logger
}

func NewEmployeeService(
	empRepo repository.EmployeeRepository,
	deptRepo repository.DepartmentRepository,
	logger *slog.Logger,
) *EmployeeService {
	return &EmployeeService{
		empRepo:  empRepo,
		deptRepo: deptRepo,
		logger:   logger.With("service", "employee"),
	}
}

func (s *EmployeeService) Create(
	departmentID uint,
	fullName, position string,
	hiredAt *time.Time,
) (*domain.Employee, error) {
	exists, err := s.deptRepo.Exists(departmentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrDepartmentNotFound
	}

	emp := &domain.Employee{
		DepartmentID: departmentID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      hiredAt,
		CreatedAt:    time.Now(),
	}

	if err := s.empRepo.Create(emp); err != nil {
		return nil, err
	}

	return emp, nil
}
