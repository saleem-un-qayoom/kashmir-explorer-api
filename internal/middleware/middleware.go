// Package middleware — Chi middleware adapters for logging, recovery, auth.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/kashmir-explorer/api/internal/config"
	pkgjwt "github.com/kashmir-explorer/api/pkg/jwt"
	"github.com/kashmir-explorer/api/pkg/response"
)

type ctxKey int

const (
	UserIDKey ctxKey = iota
	UserRoleKey
)

// Logger — structured request logger via slog.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("dur", time.Since(start)),
				slog.String("ip", r.RemoteAddr),
				slog.String("req_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}

// Recoverer — captures panics and writes a 500.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic",
						slog.Any("panic", rec),
						slog.String("stack", string(debug.Stack())),
					)
					response.Internal(w, nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Auth — extracts and verifies the Bearer JWT.
func Auth(jwt config.JWTConfig) func(http.Handler) http.Handler {
	issuer := pkgjwt.NewIssuer(jwt.Secret, jwt.RefreshSecret, jwt.AccessTTLHrs, jwt.RefreshTTLDays)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				response.Unauthorized(w, "missing or invalid Authorization header")
				return
			}
			claims, err := issuer.Verify(strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				response.Unauthorized(w, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin — gate that requires role=admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(UserRoleKey).(string)
		if role != "admin" {
			response.Forbidden(w, "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserID(r *http.Request) string {
	v, _ := r.Context().Value(UserIDKey).(string)
	return v
}
