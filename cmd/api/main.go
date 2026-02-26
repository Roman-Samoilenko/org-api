package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"org-api/internal/config"
	"org-api/internal/handler"
	"org-api/internal/middleware"
	"org-api/internal/repository"
	"org-api/internal/service"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DefaultContextTimeout = 5 * time.Second
	serverReadTimeout     = 5 * time.Second
	serverWriteTimeout    = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

// run содержит основную логику приложения и возвращает ошибку при фатальном сбое.
func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.NewConfig()
	logger.Info("configuration loaded", "addr", cfg.Addr, "port", cfg.Port)

	db, sqlDB, err := initDB(cfg.DBString)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Warn("failed to close db connection", "error", err)
		}
	}()

	if err := runMigrations(sqlDB); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	logger.Info("migrations applied successfully")

	router := buildRouter(db, logger)
	handlerWithMiddleware := middleware.Recovery(logger)(middleware.Logging(logger)(router))

	server := &http.Server{
		Addr:         cfg.Addr + ":" + cfg.Port,
		Handler:      handlerWithMiddleware,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	select {
	case <-quit:
		logger.Info("shutting down server...")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	logger.Info("server stopped gracefully")
	return nil
}

// initDB открывает соединение с БД и возвращает gorm.DB и sql.DB.
func initDB(dsn string) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}

// buildRouter инициализирует репозитории, сервисы, хендлеры и возвращает настроенный роутер.
func buildRouter(db *gorm.DB, logger *slog.Logger) *http.ServeMux {
	deptRepo := repository.NewDepartmentRepository(db)
	empRepo := repository.NewEmployeeRepository(db)

	deptService := service.NewDepartmentService(deptRepo, empRepo, db, logger)
	empService := service.NewEmployeeService(empRepo, deptRepo, logger)

	deptHandler := handler.NewDepartmentHandler(deptService, logger)
	empHandler := handler.NewEmployeeHandler(empService, logger)

	router := http.NewServeMux()
	router.HandleFunc("POST /departments", deptHandler.CreateDepartment)
	router.HandleFunc("POST /departments/{id}/employees", empHandler.CreateEmployee)
	router.HandleFunc("GET /departments/{id}", deptHandler.GetDepartment)
	router.HandleFunc("PATCH /departments/{id}", deptHandler.UpdateDepartment)
	router.HandleFunc("DELETE /departments/{id}", deptHandler.DeleteDepartment)

	return router
}

// runMigrations применяет миграции goose из папки migrations.
func runMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}
