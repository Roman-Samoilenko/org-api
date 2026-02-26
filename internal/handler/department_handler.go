package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"org-api/internal/service"
)

type DepartmentHandler struct {
	departmentService DepartmentService
	logger            *slog.Logger
}

func NewDepartmentHandler(ds DepartmentService, logger *slog.Logger) *DepartmentHandler {
	return &DepartmentHandler{
		departmentService: ds,
		logger:            logger.With("handler", "department"),
	}
}

// CreateDepartment обработчик POST /departments
func (h *DepartmentHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	defer CloseBody(h.logger, r.Body)

	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode request body", "error", err)
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	dept, err := h.departmentService.Create(req.Name, req.ParentID)
	if err != nil {
		h.handleCreateError(w, req.Name, req.ParentID, err)
		return
	}

	h.logger.Info("department created", "id", dept.ID, "name", dept.Name)
	respondJSON(w, http.StatusCreated, dept)
}

func (h *DepartmentHandler) handleCreateError(w http.ResponseWriter, name string, parentID *uint, err error) {
	switch {
	case errors.Is(err, service.ErrDuplicateName):
		h.logger.Info("duplicate department name", "name", name, "parent_id", parentID)
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrDepartmentNotFound):
		h.logger.Info("parent department not found", "parent_id", parentID)
		respondError(w, http.StatusNotFound, err.Error())
	default:
		h.logger.Error("failed to create department", "error", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
	}
}

// GetDepartment обработчик GET /departments/{id}
func (h *DepartmentHandler) GetDepartment(w http.ResponseWriter, r *http.Request) {
	id, ok := ParseID(h.logger, w, r, "id")
	if !ok {
		return
	}

	params, ok := parseGetDepartmentParams(h.logger, w, r)
	if !ok {
		return
	}

	dept, err := h.departmentService.GetWithTree(id, params.depth, params.includeEmployees, params.sortBy)
	if err != nil {
		if errors.Is(err, service.ErrDepartmentNotFound) {
			h.logger.Info("department not found", "id", id)
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		h.logger.Error("failed to get department", "id", id, "error", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondJSON(w, http.StatusOK, dept)
}

// UpdateDepartment обработчик PATCH /departments/{id}
func (h *DepartmentHandler) UpdateDepartment(w http.ResponseWriter, r *http.Request) {
	defer CloseBody(h.logger, r.Body)

	id, ok := ParseID(h.logger, w, r, "id")
	if !ok {
		return
	}

	var req UpdateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode request body", "error", err)
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.departmentService.Update(id, req.Name, req.ParentID)
	if err != nil {
		h.handleUpdateError(w, id, err)
		return
	}

	h.logger.Info("department updated", "id", updated.ID)
	respondJSON(w, http.StatusOK, updated)
}

func (h *DepartmentHandler) handleUpdateError(w http.ResponseWriter, id uint, err error) {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		h.logger.Info("department not found", "id", id)
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrDuplicateName):
		h.logger.Info("duplicate department name", "id", id)
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrCycleDetected):
		h.logger.Info("cycle detected", "id", id)
		respondError(w, http.StatusConflict, err.Error())
	default:
		h.logger.Error("failed to update department", "id", id, "error", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
	}
}

// DeleteDepartment обработчик DELETE /departments/{id}
func (h *DepartmentHandler) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, ok := ParseID(h.logger, w, r, "id")
	if !ok {
		return
	}

	deleteReq, ok := parseDeleteParams(h.logger, w, r)
	if !ok {
		return
	}

	if err := h.departmentService.Delete(id, deleteReq.Mode, deleteReq.ReassignToDepartmentID); err != nil {
		h.handleDeleteError(w, id, deleteReq.ReassignToDepartmentID, err)
		return
	}

	h.logger.Info("department deleted", "id", id, "mode", deleteReq.Mode)
	w.WriteHeader(http.StatusNoContent)
}

func (h *DepartmentHandler) handleDeleteError(w http.ResponseWriter, id uint, reassignTo *uint, err error) {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		h.logger.Info("department not found", "id", id)
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrReassignDepartmentNotFound):
		h.logger.Info("reassign department not found", "reassign_id", reassignTo)
		respondError(w, http.StatusNotFound, "reassign_to_department_id not found")
	default:
		h.logger.Error("failed to delete department", "id", id, "error", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
	}
}
