# Тестовое задание

## Старт

```bash
git clone https://github.com/Roman-Samoilenko/org-api
cd org-api
docker-compose up --build
```

Сервис доступен по адресу: **<http://localhost:8080>**

### Запуск тестов

```bash
go test ./internal/handler/... -v
```

---

## API

### POST /departments — создать подразделение

```json
// Request
{ "name": "Engineering", "parent_id": 1 }

// Response 201
{ "id": 2, "name": "Engineering", "parent_id": 1, "created_at": "..." }
```

### GET /departments/{id} — получить подразделение

Query-параметры: `depth` (1–5, по умолчанию 1), `include_employees` (bool, по умолчанию true), `sort_by` (`created_at` | `full_name`).

```json
// Response 200
{
  "id": 1, "name": "Engineering", "created_at": "...",
  "children": [{ "id": 2, "name": "Backend", "children": [] }],
  "employees": [{ "id": 1, "full_name": "Ivan Ivanov", "position": "Dev" }]
}
```

### PATCH /departments/{id} — переименовать / переместить

```json
// Request (все поля опциональны)
{ "name": "Platform", "parent_id": 3 }

// Response 200 — обновлённое подразделение
// Response 409 — дублирование имени или цикл в дереве
```

### DELETE /departments/{id} — удалить подразделение

| Query-параметр | Значение | Поведение |
|---|---|---|
| `mode` | `cascade` | Удалить подразделение, всех сотрудников и дочерние |
| `mode` | `reassign` | Удалить подразделение, сотрудников перевести в другое |
| `reassign_to_department_id` | `int` | Обязателен при `mode=reassign` |

```
Response 204 No Content
```

### POST /departments/{id}/employees — создать сотрудника

```json
// Request
{ "full_name": "Иван Иванов", "position": "Backend Engineer", "hired_at": "2023-06-01" }

// Response 201
{ "id": 1, "department_id": 2, "full_name": "Иван Иванов", "position": "Backend Engineer", "hired_at": "2023-06-01", "created_at": "..." }
```

### Коды ошибок

| Код | Причина |
|-----|---------|
| 400 | Невалидный JSON или параметры |
| 404 | Подразделение не найдено |
| 409 | Дублирование имени, цикл в дереве |
| 500 | Внутренняя ошибка сервера |

```json
{ "error": "описание ошибки" }
```

---

## Архитектура

Проект построен по принципу **Clean Architecture**. Зависимости направлены строго внутрь, бизнес-логика не знает ни о HTTP, ни о базе данных

### Слои и направление зависимостей

```mermaid
graph TD
    subgraph HTTP["HTTP Layer"]
        MW["Middleware\n(Logging, Recovery)"]
        H["Handler\n(department, employee)"]
        DTO["DTO + Validation\n(request structs)"]
    end

    subgraph SVC["Service Layer"]
        DS["DepartmentService\n· защита от циклов\n· cascade / reassign delete\n· уникальность имён"]
        ES["EmployeeService\n· проверка dept exists"]
    end

    subgraph REPO["Repository Layer"]
        DR["DepartmentRepository\n(GORM)"]
        ER["EmployeeRepository\n(GORM)"]
    end

    subgraph DOM["Domain Layer"]
        DEP["Department\nid · name · parent_id\ncreated_at"]
        EMP["Employee\nid · department_id\nfull_name · position\nhired_at · created_at"]
    end

    MW --> H
    H -->|"interface\nDepartmentService"| DS
    H -->|"interface\nEmployeeService"| ES
    DS -->|"interface\nDepartmentRepository"| DR
    DS -->|"interface\nEmployeeRepository"| ER
    ES -->|"interface\nEmployeeRepository"| ER
    ES -->|"interface\nDepartmentRepository"| DR
    DR --> DEP
    ER --> EMP
    DEP -.->|"1 — N"| EMP

    style DOM fill:#f0f4ff,stroke:#4a6cf7
    style REPO fill:#f0fff4,stroke:#38a169
    style SVC fill:#fffaf0,stroke:#d97706
    style HTTP fill:#fff0f0,stroke:#e53e3e
```

### Изоляция слоёв через интерфейсы

```mermaid
classDiagram
    direction LR

    class DepartmentHandler {
        -svc DepartmentService
        +CreateDepartment()
        +GetDepartment()
        +UpdateDepartment()
        +DeleteDepartment()
    }

    class DepartmentService {
        <<interface>>
        +Create()
        +Update()
        +GetWithTree()
        +Delete()
    }

    class DepartmentServiceImpl {
        -deptRepo DepartmentRepository
        -empRepo  EmployeeRepository
        -db       gorm.DB
        +Create()
        +Update()
        +GetWithTree()
        +Delete()
    }

    class DepartmentRepository {
        <<interface>>
        +FindByID()
        +Create()
        +Update()
        +Delete()
        +FindChildren()
        +Exists()
    }

    class departmentRepository {
        -db gorm.DB
        +FindByID()
        +Create()
        +Update()
    }

    class Department {
        +ID uint
        +Name string
        +ParentID *uint
        +CreatedAt time.Time
        +Children []Department
        +Employees []Employee
    }

    DepartmentHandler --> DepartmentService
    DepartmentService <|.. DepartmentServiceImpl
    DepartmentServiceImpl --> DepartmentRepository
    DepartmentRepository <|.. departmentRepository
    departmentRepository --> Department
```

---

## Стек

| Компонент | Технология | Роль |
|---|---|---|
| Язык | **Go 1.23** | — |
| HTTP-сервер | **net/http** (stdlib) | Маршрутизация, middleware |
| ORM | **GORM** | Запросы к БД, транзакции |
| База данных | **PostgreSQL** | Хранение данных |
| Миграции | **goose** | Версионирование схемы БД |
| Контейнеризация | **Docker + Docker Compose** | Сборка и запуск сервиса + БД |
| Логирование | **log/slog** (stdlib) | Структурированные логи |
| Тесты | **testify** + **httptest** | Юнит-тесты хендлеров с моками |

---

## Локальный запуск без Docker

```bash
# 1. Поднять PostgreSQL и создать базу org-db

# 2. Задать переменные окружения
export db_string="postgres://postgres:postgres@localhost:5432/org-db?sslmode=disable"

# 3. Запустить
go run cmd/api/main.go
```
