package repository

import (
	"errors"

	"org-api/internal/domain"

	"gorm.io/gorm"
)

type DepartmentRepository interface {
	Create(dept *domain.Department) error
	Update(dept *domain.Department) error
	Delete(id uint) error
	FindByID(id uint) (*domain.Department, error)
	FindByNameAndParent(name string, parentID *uint) (*domain.Department, error)
	ExistsByNameAndParent(name string, parentID *uint) (bool, error)
	FindChildren(parentID uint) ([]domain.Department, error)
	Exists(id uint) (bool, error)
}

type departmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) DepartmentRepository {
	return &departmentRepository{db: db}
}

func (r *departmentRepository) Create(dept *domain.Department) error {
	return r.db.Create(dept).Error
}

func (r *departmentRepository) Update(dept *domain.Department) error {
	return r.db.Save(dept).Error
}

func (r *departmentRepository) Delete(id uint) error {
	return r.db.Delete(&domain.Department{}, id).Error
}

func (r *departmentRepository) FindByID(id uint) (*domain.Department, error) {
	var dept domain.Department
	err := r.db.First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) FindByNameAndParent(name string, parentID *uint) (*domain.Department, error) {
	var dept domain.Department
	query := r.db.Where("name = ? AND (parent_id = ? OR (parent_id IS NULL AND ? IS NULL))", name, parentID, parentID)
	err := query.First(&dept).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) ExistsByNameAndParent(name string, parentID *uint) (bool, error) {
	var count int64
	query := r.db.Model(&domain.Department{}).
		Where("name = ? AND (parent_id = ? OR (parent_id IS NULL AND ? IS NULL))", name, parentID, parentID)
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *departmentRepository) FindChildren(parentID uint) ([]domain.Department, error) {
	var children []domain.Department
	err := r.db.Where("parent_id = ?", parentID).Find(&children).Error
	return children, err
}

func (r *departmentRepository) Exists(id uint) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Department{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}
