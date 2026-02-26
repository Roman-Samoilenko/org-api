package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"org-api/internal/service"
)

type EmployeeHandler struct {
	employeeService EmployeeService
	logger          *slog.Logger
}

func NewEmployeeHandler(es EmployeeService, logger *slog.Logger) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService: es,
		logger:          logger.With("handler", "employee"),
	}
}

// CreateEmployee обработчик POST /departments/{id}/employees.
func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	deptIDStr := r.PathValue("id")
	deptID, err := strconv.Atoi(deptIDStr)
	if err != nil {
		h.logger.Warn("invalid department ID", "id", deptIDStr)
		respondError(w, http.StatusBadRequest, "Invalid department ID")
		return
	}

	var req CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode request body", "error", err)
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	defer func() {
		err := r.Body.Close()
		if err != nil {
			h.logger.Warn("failed to close request body", "error", err)
		}
	}()

	if err := req.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var hiredAt *time.Time
	if req.HiredAt != nil {
		t, err := time.Parse(DateFormat, *req.HiredAt)
		if err != nil {
			h.logger.Error("unexpected hired_at parse error", "error", err)
			respondError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		hiredAt = &t
	}

	emp, err := h.employeeService.Create(uint(deptID), req.FullName, req.Position, hiredAt)
	if err != nil {
		if errors.Is(err, service.ErrDepartmentNotFound) {
			h.logger.Info("department not found", "department_id", deptID)
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		h.logger.Error("failed to create employee", "error", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	h.logger.Info("employee created", "id", emp.ID, "full_name", emp.FullName)
	respondJSON(w, http.StatusCreated, emp)
}
