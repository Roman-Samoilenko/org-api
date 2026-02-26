package repository

import (
	"org-api/internal/domain"

	"gorm.io/gorm"
)

type EmployeeRepository interface {
	Create(emp *domain.Employee) error
	DeleteByDepartment(departmentID uint) error
	UpdateDepartment(employeeIDs []uint, newDepartmentID uint) error
	FindByDepartment(departmentID uint) ([]domain.Employee, error)
	FindByDepartmentOrdered(departmentID uint, orderBy string) ([]domain.Employee, error)
}

type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) Create(emp *domain.Employee) error {
	return r.db.Create(emp).Error
}

func (r *employeeRepository) DeleteByDepartment(departmentID uint) error {
	return r.db.Where("department_id = ?", departmentID).Delete(&domain.Employee{}).Error
}

func (r *employeeRepository) UpdateDepartment(employeeIDs []uint, newDepartmentID uint) error {
	return r.db.Model(&domain.Employee{}).
		Where("id IN ?", employeeIDs).
		Update("department_id", newDepartmentID).
		Error
}

func (r *employeeRepository) FindByDepartment(departmentID uint) ([]domain.Employee, error) {
	var employees []domain.Employee
	err := r.db.Where("department_id = ?", departmentID).Find(&employees).Error
	return employees, err
}

func (r *employeeRepository) FindByDepartmentOrdered(
	departmentID uint,
	orderBy string,
) ([]domain.Employee, error) {
	var employees []domain.Employee
	err := r.db.Where("department_id = ?", departmentID).Order(orderBy).Find(&employees).Error
	return employees, err
}
