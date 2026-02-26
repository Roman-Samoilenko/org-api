package handler_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"org-api/internal/domain"
	"org-api/internal/handler"
	"org-api/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Мок сервиса подразделений

type mockDepartmentService struct {
	createFn      func(name string, parentID *uint) (*domain.Department, error)
	updateFn      func(id uint, name *string, parentID *uint) (*domain.Department, error)
	getWithTreeFn func(id uint, depth int, includeEmployees bool, sortBy string) (*domain.Department, error)
	deleteFn      func(id uint, mode string, reassignTo *uint) error
}

func (m *mockDepartmentService) Create(name string, parentID *uint) (*domain.Department, error) {
	return m.createFn(name, parentID)
}

func (m *mockDepartmentService) Update(
	id uint,
	name *string,
	parentID *uint,
) (*domain.Department, error) {
	return m.updateFn(id, name, parentID)
}

func (m *mockDepartmentService) GetWithTree(
	id uint,
	depth int,
	includeEmployees bool,
	sortBy string,
) (*domain.Department, error) {
	return m.getWithTreeFn(id, depth, includeEmployees, sortBy)
}
func (m *mockDepartmentService) Delete(id uint, mode string, reassignTo *uint) error {
	return m.deleteFn(id, mode, reassignTo)
}

// Хелперы

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newDeptHandler(svc handler.DepartmentService) *handler.DepartmentHandler {
	return handler.NewDepartmentHandler(svc, testLogger())
}

func ptr[T any](v T) *T { return &v }

func makeDept(id uint, name string, parentID *uint) *domain.Department {
	return &domain.Department{
		ID:        id,
		Name:      name,
		ParentID:  parentID,
		CreatedAt: time.Now(),
		Children:  []domain.Department{},
		Employees: []domain.Employee{},
	}
}

// decodeBody декодирует тело ответа в map — позволяет избежать ошибки musttag
// (линтер требует json-теги на struct, которые используются с json.Decode).
func decodeBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

// CreateDepartment

func TestCreateDepartment_Success(t *testing.T) {
	svc := &mockDepartmentService{
		createFn: func(name string, parentID *uint) (*domain.Department, error) {
			return makeDept(1, name, parentID), nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/departments", jsonBody(`{"name":"Engineering"}`))
	w := httptest.NewRecorder()
	h.CreateDepartment(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := decodeBody(t, w.Body.Bytes())
	assert.Equal(t, "Engineering", resp["name"])
}

func TestCreateDepartment_WithParentID(t *testing.T) {
	svc := &mockDepartmentService{
		createFn: func(name string, parentID *uint) (*domain.Department, error) {
			return makeDept(2, name, parentID), nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/departments",
		jsonBody(`{"name":"Backend","parent_id":1}`),
	)
	w := httptest.NewRecorder()
	h.CreateDepartment(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateDepartment_InvalidJSON(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBufferString(`not-json`))
	w := httptest.NewRecorder()
	h.CreateDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorContains(t, w.Body.Bytes(), "Invalid JSON")
}

func TestCreateDepartment_EmptyName(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodPost, "/departments", jsonBody(`{"name":"   "}`))
	w := httptest.NewRecorder()
	h.CreateDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDepartment_DuplicateName(t *testing.T) {
	svc := &mockDepartmentService{
		createFn: func(_ string, _ *uint) (*domain.Department, error) {
			return nil, service.ErrDuplicateName
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/departments", jsonBody(`{"name":"Engineering"}`))
	w := httptest.NewRecorder()
	h.CreateDepartment(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateDepartment_ParentNotFound(t *testing.T) {
	svc := &mockDepartmentService{
		createFn: func(_ string, _ *uint) (*domain.Department, error) {
			return nil, service.ErrDepartmentNotFound
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/departments",
		jsonBody(`{"name":"Backend","parent_id":999}`),
	)
	w := httptest.NewRecorder()
	h.CreateDepartment(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// GetDepartment

func TestGetDepartment_Success(t *testing.T) {
	svc := &mockDepartmentService{
		getWithTreeFn: func(id uint, depth int, includeEmployees bool, sortBy string) (*domain.Department, error) {
			assert.Equal(t, uint(1), id)
			assert.Equal(t, 1, depth)
			assert.True(t, includeEmployees)
			assert.Equal(t, "created_at", sortBy)
			return makeDept(1, "Engineering", nil), nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/departments/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.GetDepartment(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetDepartment_WithQueryParams(t *testing.T) {
	svc := &mockDepartmentService{
		getWithTreeFn: func(_ uint, depth int, includeEmployees bool, sortBy string) (*domain.Department, error) {
			assert.Equal(t, 3, depth)
			assert.False(t, includeEmployees)
			assert.Equal(t, "full_name", sortBy)
			return makeDept(1, "Engineering", nil), nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(
		http.MethodGet,
		"/departments/1?depth=3&include_employees=false&sort_by=full_name",
		nil,
	)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.GetDepartment(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetDepartment_NotFound(t *testing.T) {
	svc := &mockDepartmentService{
		getWithTreeFn: func(_ uint, _ int, _ bool, _ string) (*domain.Department, error) {
			return nil, service.ErrDepartmentNotFound
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/departments/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()
	h.GetDepartment(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetDepartment_InvalidID(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodGet, "/departments/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	h.GetDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDepartment_InvalidDepth(t *testing.T) {
	for _, depth := range []string{"0", "6", "abc"} {
		t.Run("depth="+depth, func(t *testing.T) {
			h := newDeptHandler(&mockDepartmentService{})
			req := httptest.NewRequest(http.MethodGet, "/departments/1?depth="+depth, nil)
			req.SetPathValue("id", "1")
			w := httptest.NewRecorder()
			h.GetDepartment(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestGetDepartment_InvalidSortBy(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodGet, "/departments/1?sort_by=unknown", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.GetDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// UpdateDepartment

func TestUpdateDepartment_Success(t *testing.T) {
	svc := &mockDepartmentService{
		updateFn: func(id uint, name *string, parentID *uint) (*domain.Department, error) {
			return makeDept(id, *name, parentID), nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/departments/1", jsonBody(`{"name":"Platform"}`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.UpdateDepartment(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeBody(t, w.Body.Bytes())
	assert.Equal(t, "Platform", resp["name"])
}

func TestUpdateDepartment_NotFound(t *testing.T) {
	svc := &mockDepartmentService{
		updateFn: func(_ uint, _ *string, _ *uint) (*domain.Department, error) {
			return nil, service.ErrDepartmentNotFound
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/departments/999", jsonBody(`{"name":"X"}`))
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()
	h.UpdateDepartment(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateDepartment_CycleDetected(t *testing.T) {
	svc := &mockDepartmentService{
		updateFn: func(_ uint, _ *string, _ *uint) (*domain.Department, error) {
			return nil, service.ErrCycleDetected
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/departments/1", jsonBody(`{"parent_id":2}`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.UpdateDepartment(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateDepartment_DuplicateName(t *testing.T) {
	svc := &mockDepartmentService{
		updateFn: func(_ uint, _ *string, _ *uint) (*domain.Department, error) {
			return nil, service.ErrDuplicateName
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodPatch, "/departments/1", jsonBody(`{"name":"Backend"}`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.UpdateDepartment(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateDepartment_EmptyNameValidation(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodPatch, "/departments/1", jsonBody(`{"name":""}`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.UpdateDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// DeleteDepartment

func TestDeleteDepartment_CascadeSuccess(t *testing.T) {
	svc := &mockDepartmentService{
		deleteFn: func(id uint, mode string, reassignTo *uint) error {
			assert.Equal(t, uint(1), id)
			assert.Equal(t, "cascade", mode)
			assert.Nil(t, reassignTo)
			return nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/departments/1?mode=cascade", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.DeleteDepartment(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteDepartment_ReassignSuccess(t *testing.T) {
	svc := &mockDepartmentService{
		deleteFn: func(_ uint, mode string, reassignTo *uint) error {
			assert.Equal(t, "reassign", mode)
			require.NotNil(t, reassignTo)
			assert.Equal(t, uint(2), *reassignTo)
			return nil
		},
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/departments/1?mode=reassign&reassign_to_department_id=2",
		nil,
	)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.DeleteDepartment(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteDepartment_InvalidMode(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodDelete, "/departments/1?mode=wrong", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.DeleteDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteDepartment_ReassignWithoutTarget(t *testing.T) {
	h := newDeptHandler(&mockDepartmentService{})

	req := httptest.NewRequest(http.MethodDelete, "/departments/1?mode=reassign", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.DeleteDepartment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteDepartment_NotFound(t *testing.T) {
	svc := &mockDepartmentService{
		deleteFn: func(_ uint, _ string, _ *uint) error { return service.ErrDepartmentNotFound },
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/departments/999?mode=cascade", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()
	h.DeleteDepartment(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteDepartment_ReassignTargetNotFound(t *testing.T) {
	svc := &mockDepartmentService{
		deleteFn: func(_ uint, _ string, _ *uint) error { return service.ErrReassignDepartmentNotFound },
	}
	h := newDeptHandler(svc)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/departments/1?mode=reassign&reassign_to_department_id=999",
		nil,
	)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.DeleteDepartment(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Хелперы

func jsonBody(s string) *bytes.Buffer { return bytes.NewBufferString(s) }

func assertErrorContains(t *testing.T, body []byte, substr string) {
	t.Helper()
	var resp map[string]string
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Contains(t, resp["error"], substr)
}
