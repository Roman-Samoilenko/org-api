-- +goose Up
CREATE TABLE employees (
    id            SERIAL PRIMARY KEY,
    department_id INTEGER NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    full_name     VARCHAR(200) NOT NULL,
    position      VARCHAR(200) NOT NULL,
    hired_at      DATE,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_employees_department_id ON employees(department_id);
CREATE INDEX idx_employees_hired_at ON employees(hired_at);
CREATE INDEX idx_employees_created_at ON employees(created_at);

-- +goose Down
DROP TABLE employees;