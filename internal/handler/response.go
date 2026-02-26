package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// respondJSON отправляет успешный ответ в формате JSON.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// В случае ошибки кодирования уже ничего не отправить, но можно залогировать
		slog.Error("failed to encode response", "error", err)
	}
}

// respondError отправляет ошибку в формате JSON.
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
