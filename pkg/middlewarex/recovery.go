package middlewarex

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"go-backend-example/pkg/logx"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		defer func() {
			if rec := recover(); rec != nil {
				logger(ctx).Error(
					"panic in handler",
					slog.Any(logx.FieldError, rec),
					slog.String(logx.FieldStack, string(debug.Stack())),
				)

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
