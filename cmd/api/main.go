package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"org-api/internal/config"
	"org-api/internal/handler"
	"org-api/internal/middleware"
	"org-api/internal/repository"
	"org-api/internal/service"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.NewConfig()
	logger.Info("configuration loaded", "addr", cfg.Addr, "port", cfg.Port)

	db, sqlDB, err := initDB(cfg.DBString)
	if err != nil {
		logger.Error("failed to init database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Warn("failed to close db connection", "error", err)
		}
	}()

	if err := runMigrations(sqlDB); err != nil {
		logger.Error("failed to apply migrations", "error", err)
		os.Exit(1)
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

	go func() {
		logger.Info("starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped gracefully")
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
