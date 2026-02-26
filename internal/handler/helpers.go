package handler

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// CloseBody безопасно закрывает тело запроса и логирует ошибку.
func CloseBody(logger *slog.Logger, body io.ReadCloser) {
	if err := body.Close(); err != nil {
		logger.Warn("failed to close request body", "error", err)
	}
}

// ParseID извлекает и парсит ID из path-параметра.
func ParseID(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	paramName string,
) (uint, bool) {
	idStr := r.PathValue(paramName)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Warn("invalid ID", "param", paramName, "value", idStr)
		respondError(w, http.StatusBadRequest, "Invalid "+paramName+" ID")
		return 0, false
	}
	return uint(id), true
}

// ParseDepth парсит параметр depth с валидацией.
func ParseDepth(logger *slog.Logger, w http.ResponseWriter, r *http.Request) (int, bool) {
	depthStr := r.URL.Query().Get("depth")
	if depthStr == "" {
		return 1, true
	}
	depth, err := strconv.Atoi(depthStr)
	if err != nil || depth < 1 || depth > 5 {
		logger.Warn("invalid depth parameter", "value", depthStr)
		respondError(w, http.StatusBadRequest, "depth must be an integer between 1 and 5")
		return 0, false
	}
	return depth, true
}

// ParseBoolQuery парсит boolean query-параметр с дефолтным значением.
func ParseBoolQuery(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	paramName string,
	defaultValue bool,
) (bool, bool) {
	valueStr := r.URL.Query().Get(paramName)
	if valueStr == "" {
		return defaultValue, true
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		logger.Warn("invalid boolean parameter", "param", paramName, "value", valueStr)
		respondError(w, http.StatusBadRequest, paramName+" must be a boolean")
		return false, false
	}
	return value, true
}

// getDepartmentParams — параметры запроса GET /departments/{id}.
type getDepartmentParams struct {
	sortBy           string
	depth            int
	includeEmployees bool
}

// parseGetDepartmentParams парсит и валидирует все query-параметры GET /departments/{id}.
func parseGetDepartmentParams(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
) (getDepartmentParams, bool) {
	depth, ok := ParseDepth(logger, w, r)
	if !ok {
		return getDepartmentParams{}, false
	}

	includeEmployees, ok := ParseBoolQuery(logger, w, r, "include_employees", true)
	if !ok {
		return getDepartmentParams{}, false
	}

	sortBy, ok := parseSortBy(logger, w, r)
	if !ok {
		return getDepartmentParams{}, false
	}

	return getDepartmentParams{
		depth:            depth,
		includeEmployees: includeEmployees,
		sortBy:           sortBy,
	}, true
}

// parseSortBy парсит параметр sort_by: допустимые значения "created_at" и "full_name".
func parseSortBy(logger *slog.Logger, w http.ResponseWriter, r *http.Request) (string, bool) {
	s := r.URL.Query().Get("sort_by")
	switch s {
	case "", "created_at":
		return "created_at", true
	case "full_name":
		return "full_name", true
	default:
		logger.Warn("invalid sort_by parameter", "value", s)
		respondError(w, http.StatusBadRequest, "sort_by must be either 'created_at' or 'full_name'")
		return "", false
	}
}

// parseDeleteParams парсит query-параметры DELETE /departments/{id} и валидирует их.
func parseDeleteParams(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
) (DeleteDepartmentRequest, bool) {
	query := r.URL.Query()
	mode := query.Get("mode")
	reassignToStr := query.Get("reassign_to_department_id")

	var reassignTo *uint
	if reassignToStr != "" {
		val, err := strconv.Atoi(reassignToStr)
		if err != nil {
			logger.Warn("invalid reassign_to_department_id", "value", reassignToStr)
			respondError(w, http.StatusBadRequest, "reassign_to_department_id must be an integer")
			return DeleteDepartmentRequest{}, false
		}
		u := uint(val)
		reassignTo = &u
	}

	req := DeleteDepartmentRequest{
		Mode:                   mode,
		ReassignToDepartmentID: reassignTo,
	}
	if err := req.Validate(); err != nil {
		logger.Warn("validation failed", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return DeleteDepartmentRequest{}, false
	}

	return req, true
}
