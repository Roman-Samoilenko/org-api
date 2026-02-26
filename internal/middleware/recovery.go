package middleware

import (
	"log/slog"
	"net/http"
)

// Recovery возвращает middleware для восстановления после паники.
func Recovery(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"error", rec,
						"method", r.Method,
						"path", r.URL.Path,
					)
					w.WriteHeader(http.StatusInternalServerError)
					if _, err := w.Write([]byte("Internal Server Error")); err != nil {
						logger.Warn("failed to write recovery response", "error", err)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
