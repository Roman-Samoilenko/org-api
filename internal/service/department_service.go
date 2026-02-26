package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"org-api/internal/domain"
	"org-api/internal/repository"

	"gorm.io/gorm"
)

type DepartmentService struct {
	deptRepo repository.DepartmentRepository
	empRepo  repository.EmployeeRepository
	db       *gorm.DB
	logger   *slog.Logger
}

func NewDepartmentService(
	deptRepo repository.DepartmentRepository,
	empRepo repository.EmployeeRepository,
	db *gorm.DB,
	logger *slog.Logger,
) *DepartmentService {
	return &DepartmentService{
		deptRepo: deptRepo,
		empRepo:  empRepo,
		db:       db,
		logger:   logger.With("service", "department"),
	}
}

func (s *DepartmentService) Create(name string, parentID *uint) (*domain.Department, error) {
	exists, err := s.deptRepo.ExistsByNameAndParent(name, parentID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateName
	}

	if parentID != nil {
		if err := s.assertDepartmentExists(*parentID); err != nil {
			return nil, err
		}
	}

	dept := &domain.Department{
		Name:      name,
		ParentID:  parentID,
		CreatedAt: time.Now(),
	}

	if err := s.deptRepo.Create(dept); err != nil {
		return nil, err
	}

	return dept, nil
}

// Update обновляет имя и/или родителя подразделения.
func (s *DepartmentService) Update(
	id uint,
	name *string,
	parentID *uint,
) (*domain.Department, error) {
	dept, err := s.deptRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	if err := s.applyNameUpdate(dept, name, parentID); err != nil {
		return nil, err
	}

	if err := s.applyParentUpdate(dept, id, parentID); err != nil {
		return nil, err
	}

	if err := s.deptRepo.Update(dept); err != nil {
		return nil, err
	}

	return dept, nil
}

// GetWithTree возвращает подразделение с деревом дочерних и (опционально) сотрудниками.
// sortBy задаёт поле сортировки сотрудников: "created_at" или "full_name".
func (s *DepartmentService) GetWithTree(
	id uint,
	depth int,
	includeEmployees bool,
	sortBy string,
) (*domain.Department, error) {
	root, err := s.deptRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	children, err := s.getSubtree(id, depth-1)
	if err != nil {
		return nil, err
	}
	root.Children = children

	if includeEmployees {
		if sortBy == "" {
			sortBy = "created_at"
		}
		employees, err := s.empRepo.FindByDepartmentOrdered(id, sortBy)
		if err != nil {
			return nil, err
		}
		root.Employees = employees
	}

	return root, nil
}

// Delete удаляет подразделение в режиме cascade или reassign.
func (s *DepartmentService) Delete(id uint, mode string, reassignTo *uint) error {
	if _, err := s.deptRepo.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDepartmentNotFound
		}
		return err
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	allIDs, err := s.collectAllIDs(tx, id)
	if err != nil {
		return err
	}

	if err := s.executeDelete(tx, allIDs, mode, reassignTo); err != nil {
		return err
	}

	return tx.Commit().Error
}

// executeDelete выполняет удаление или переназначение в рамках транзакции.
func (s *DepartmentService) executeDelete(
	tx *gorm.DB,
	allIDs []uint,
	mode string,
	reassignTo *uint,
) error {
	switch mode {
	case "cascade":
		return s.deleteCascade(tx, allIDs)
	case "reassign":
		return s.deleteReassign(tx, allIDs, reassignTo)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

func (s *DepartmentService) deleteCascade(tx *gorm.DB, allIDs []uint) error {
	if err := tx.Where("department_id IN ?", allIDs).Delete(&domain.Employee{}).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", allIDs).Delete(&domain.Department{}).Error
}

func (s *DepartmentService) deleteReassign(tx *gorm.DB, allIDs []uint, reassignTo *uint) error {
	if reassignTo == nil {
		return errors.New("reassign target required")
	}

	var count int64
	if err := tx.Model(&domain.Department{}).Where("id = ?", *reassignTo).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrReassignDepartmentNotFound
	}

	if err := tx.Model(&domain.Employee{}).
		Where("department_id IN ?", allIDs).
		Update("department_id", *reassignTo).Error; err != nil {
		return err
	}

	return tx.Where("id IN ?", allIDs).Delete(&domain.Department{}).Error
}

// collectAllIDs возвращает id самого подразделения и всех его потомков.
func (s *DepartmentService) collectAllIDs(tx *gorm.DB, id uint) ([]uint, error) {
	allIDs := []uint{id}
	subIDs, err := s.collectDescendantIDs(tx, id)
	if err != nil {
		return nil, err
	}
	return append(allIDs, subIDs...), nil
}

// assertDepartmentExists возвращает ErrDepartmentNotFound, если подразделение не существует.
func (s *DepartmentService) assertDepartmentExists(id uint) error {
	exists, err := s.deptRepo.Exists(id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrDepartmentNotFound
	}
	return nil
}

// isDescendant проверяет, является ли descendantID потомком ancestorID.
func (s *DepartmentService) isDescendant(descendantID, ancestorID uint) (bool, error) {
	currentID := descendantID
	for {
		dept, err := s.deptRepo.FindByID(currentID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return false, nil
			}
			return false, err
		}
		if dept.ParentID == nil {
			return false, nil
		}
		if *dept.ParentID == ancestorID {
			return true, nil
		}
		currentID = *dept.ParentID
	}
}

// getSubtree рекурсивно строит дерево дочерних подразделений до заданной глубины.
func (s *DepartmentService) getSubtree(parentID uint, depth int) ([]domain.Department, error) {
	children, err := s.deptRepo.FindChildren(parentID)
	if err != nil {
		return nil, err
	}

	if depth <= 0 {
		return children, nil
	}

	for i := range children {
		grandChildren, err := s.getSubtree(children[i].ID, depth-1)
		if err != nil {
			return nil, err
		}
		children[i].Children = grandChildren
	}

	return children, nil
}

// collectDescendantIDs рекурсивно собирает ID всех потомков parentID.
func (s *DepartmentService) collectDescendantIDs(tx *gorm.DB, parentID uint) ([]uint, error) {
	var children []struct{ ID uint }
	if err := tx.Table("departments").Select("id").Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return nil, err
	}

	var ids []uint
	for _, child := range children {
		ids = append(ids, child.ID)
		subIDs, err := s.collectDescendantIDs(tx, child.ID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, subIDs...)
	}
	return ids, nil
}

// applyNameUpdate проверяет уникальность нового имени и применяет его к dept.
func (s *DepartmentService) applyNameUpdate(
	dept *domain.Department,
	name *string,
	newParentID *uint,
) error {
	if name == nil {
		return nil
	}

	// Берём эффективного родителя: если меняется parent — используем новый.
	effectiveParent := dept.ParentID
	if newParentID != nil {
		effectiveParent = newParentID
	}

	exists, err := s.deptRepo.ExistsByNameAndParent(*name, effectiveParent)
	if err != nil {
		return err
	}

	if exists {
		existing, err := s.deptRepo.FindByNameAndParent(*name, effectiveParent)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != dept.ID {
			return ErrDuplicateName
		}
	}

	dept.Name = *name
	return nil
}

// applyParentUpdate проверяет корректность нового родителя и применяет его к dept.
func (s *DepartmentService) applyParentUpdate(
	dept *domain.Department,
	id uint,
	parentID *uint,
) error {
	if parentID == nil {
		return nil
	}

	if *parentID == id {
		return ErrCycleDetected
	}

	if err := s.assertDepartmentExists(*parentID); err != nil {
		return err
	}

	isDesc, err := s.isDescendant(*parentID, id)
	if err != nil {
		return err
	}
	if isDesc {
		return ErrCycleDetected
	}

	dept.ParentID = parentID
	return nil
}
