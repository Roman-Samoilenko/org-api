package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"org-api/internal/domain"
	"org-api/internal/handler"
	"org-api/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Мок сервиса сотрудников

type mockEmployeeService struct {
	createFn func(departmentID uint, fullName, position string, hiredAt *time.Time) (*domain.Employee, error)
}

func (m *mockEmployeeService) Create(
	departmentID uint,
	fullName, position string,
	hiredAt *time.Time,
) (*domain.Employee, error) {
	return m.createFn(departmentID, fullName, position, hiredAt)
}

func newEmpHandler(svc handler.EmployeeService) *handler.EmployeeHandler {
	return handler.NewEmployeeHandler(svc, testLogger())
}

// CreateEmployee

func TestCreateEmployee_Success(t *testing.T) {
	svc := &mockEmployeeService{
		createFn: func(departmentID uint, fullName, position string, hiredAt *time.Time) (*domain.Employee, error) {
			assert.Equal(t, uint(1), departmentID)
			assert.Equal(t, "Ivan Ivanov", fullName)
			assert.Equal(t, "Engineer", position)
			assert.Nil(t, hiredAt)
			return &domain.Employee{
				ID:           1,
				DepartmentID: departmentID,
				FullName:     fullName,
				Position:     position,
				CreatedAt:    time.Now(),
			}, nil
		},
	}
	h := newEmpHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/departments/1/employees",
		jsonBody(`{"full_name":"Ivan Ivanov","position":"Engineer"}`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.CreateEmployee(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := decodeBody(t, w.Body.Bytes())
	assert.Equal(t, "Ivan Ivanov", resp["full_name"])
	assert.Equal(t, "Engineer", resp["position"])
}

func TestCreateEmployee_WithHiredAt(t *testing.T) {
	svc := &mockEmployeeService{
		createFn: func(_ uint, _, _ string, hiredAt *time.Time) (*domain.Employee, error) {
			require.NotNil(t, hiredAt)
			assert.Equal(t, 2023, hiredAt.Year())
			assert.Equal(t, time.January, hiredAt.Month())
			assert.Equal(t, 15, hiredAt.Day())
			return &domain.Employee{
				ID:       1,
				FullName: "Anna",
				Position: "Manager",
				HiredAt:  hiredAt,
			}, nil
		},
	}
	h := newEmpHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/departments/1/employees",
		jsonBody(`{"full_name":"Anna Petrova","position":"Manager","hired_at":"2023-01-15"}`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.CreateEmployee(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateEmployee_DepartmentNotFound(t *testing.T) {
	svc := &mockEmployeeService{
		createFn: func(_ uint, _, _ string, _ *time.Time) (*domain.Employee, error) {
			return nil, service.ErrDepartmentNotFound
		},
	}
	h := newEmpHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/departments/999/employees",
		jsonBody(`{"full_name":"Ivan","position":"Dev"}`))
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()
	h.CreateEmployee(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateEmployee_InvalidDepartmentID(t *testing.T) {
	h := newEmpHandler(&mockEmployeeService{})

	req := httptest.NewRequest(http.MethodPost, "/departments/abc/employees", jsonBody(`{}`))
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	h.CreateEmployee(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEmployee_InvalidJSON(t *testing.T) {
	h := newEmpHandler(&mockEmployeeService{})

	req := httptest.NewRequest(http.MethodPost, "/departments/1/employees", jsonBody(`not-json`))
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.CreateEmployee(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEmployee_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty full_name", `{"full_name":"","position":"Engineer"}`},
		{"missing full_name", `{"position":"Engineer"}`},
		{"empty position", `{"full_name":"Ivan","position":""}`},
		{"missing position", `{"full_name":"Ivan"}`},
		{
			"invalid hired_at format",
			`{"full_name":"Ivan","position":"Dev","hired_at":"15-01-2023"}`,
		},
		{"empty hired_at string", `{"full_name":"Ivan","position":"Dev","hired_at":""}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newEmpHandler(&mockEmployeeService{})
			req := httptest.NewRequest(
				http.MethodPost,
				"/departments/1/employees",
				jsonBody(tc.body),
			)
			req.SetPathValue("id", "1")
			w := httptest.NewRecorder()
			h.CreateEmployee(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// DTO Validation

func TestCreateDepartmentRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		req     handler.CreateDepartmentRequest
		wantErr bool
	}{
		{"valid", handler.CreateDepartmentRequest{Name: "Engineering"}, false},
		{"trims spaces", handler.CreateDepartmentRequest{Name: "  Engineering  "}, false},
		{"empty name", handler.CreateDepartmentRequest{Name: ""}, true},
		{"only spaces", handler.CreateDepartmentRequest{Name: "   "}, true},
		{"name too long", handler.CreateDepartmentRequest{Name: string(make([]byte, 201))}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateEmployeeRequest_Validate(t *testing.T) {
	validHiredAt := "2023-06-01"
	cases := []struct {
		wantErr bool
		name    string
		req     handler.CreateEmployeeRequest
	}{
		{
			false,
			"valid without hired_at",
			handler.CreateEmployeeRequest{FullName: "Ivan", Position: "Dev"},
		},
		{
			false,
			"valid with hired_at",
			handler.CreateEmployeeRequest{
				FullName: "Ivan",
				Position: "Dev",
				HiredAt:  &validHiredAt,
			},
		},
		{true, "empty full_name", handler.CreateEmployeeRequest{FullName: "", Position: "Dev"}},
		{true, "empty position", handler.CreateEmployeeRequest{FullName: "Ivan", Position: ""}},
		{
			true,
			"invalid hired_at",
			handler.CreateEmployeeRequest{
				FullName: "Ivan",
				Position: "Dev",
				HiredAt:  ptr("bad-date"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateDepartmentRequest_Validate(t *testing.T) {
	cases := []struct {
		wantErr bool
		name    string
		req     handler.UpdateDepartmentRequest
	}{
		{false, "nil name (no change)", handler.UpdateDepartmentRequest{Name: nil}},
		{false, "valid name", handler.UpdateDepartmentRequest{Name: ptr("Platform")}},
		{true, "empty name", handler.UpdateDepartmentRequest{Name: ptr("")}},
		{true, "only spaces", handler.UpdateDepartmentRequest{Name: ptr("   ")}},
		{true, "name too long", handler.UpdateDepartmentRequest{Name: ptr(string(make([]byte, 201)))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteDepartmentRequest_Validate(t *testing.T) {
	reassignID := uint(2)
	cases := []struct {
		name    string
		req     handler.DeleteDepartmentRequest
		wantErr bool
	}{
		{"cascade", handler.DeleteDepartmentRequest{Mode: "cascade"}, false},
		{
			"reassign with target",
			handler.DeleteDepartmentRequest{Mode: "reassign", ReassignToDepartmentID: &reassignID},
			false,
		},
		{"invalid mode", handler.DeleteDepartmentRequest{Mode: "drop"}, true},
		{"reassign without target", handler.DeleteDepartmentRequest{Mode: "reassign"}, true},
		{"empty mode", handler.DeleteDepartmentRequest{Mode: ""}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
