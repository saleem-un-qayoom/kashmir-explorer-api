// Package response — consistent JSON envelope helpers.
package response

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Code  string `json:"code,omitempty"`
	Meta  any    `json:"meta,omitempty"`
}

func OK(w http.ResponseWriter, data any) {
	write(w, http.StatusOK, Envelope{Data: data})
}

func Created(w http.ResponseWriter, data any) {
	write(w, http.StatusCreated, Envelope{Data: data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func BadRequest(w http.ResponseWriter, msg string) {
	write(w, http.StatusBadRequest, Envelope{Error: msg, Code: "bad_request"})
}

func Unauthorized(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "authentication required"
	}
	write(w, http.StatusUnauthorized, Envelope{Error: msg, Code: "unauthorized"})
}

func Forbidden(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "forbidden"
	}
	write(w, http.StatusForbidden, Envelope{Error: msg, Code: "forbidden"})
}

func NotFound(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "not found"
	}
	write(w, http.StatusNotFound, Envelope{Error: msg, Code: "not_found"})
}

// Error writes an arbitrary status with a machine-readable code and message.
// Use the named helpers (BadRequest, NotFound, …) for the common cases; this
// is for statuses without a dedicated helper, e.g. 503 Service Unavailable.
func Error(w http.ResponseWriter, status int, code, msg string) {
	write(w, status, Envelope{Error: msg, Code: code})
}

// Internal logs the full error server-side and returns a generic body, never
// leaking the underlying error text (SQL/driver detail) to API consumers.
//
// Deprecated: prefer FromError, which additionally maps domain sentinels
// (pgx.ErrNoRows → 404, unique violation → 409) before falling back here.
// Kept so handlers not yet migrated to FromError still avoid leaking.
func Internal(w http.ResponseWriter, err error) {
	internal(context.Background(), w, err)
}

// FromError is the central error mapper. It maps known domain sentinels to
// their HTTP status and writes a generic body for everything else, logging the
// full error server-side. Unexpected errors never reach the client verbatim.
//
//   - pgx.ErrNoRows                       → 404 not_found
//   - unique violation (SQLSTATE 23505)   → 409 conflict
//   - anything else                       → 500 internal (generic body, logged)
//
// notFoundMsg customizes the 404 body (e.g. "destination not found"); pass ""
// for the default "not found".
func FromError(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		NotFound(w, notFoundMsg)
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		write(w, http.StatusConflict, Envelope{Error: "resource already exists", Code: "conflict"})
		return
	}

	internal(reqContext(r), w, err)
}

// internal logs the full error and writes the generic 500 body.
func internal(ctx context.Context, w http.ResponseWriter, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "unexpected handler error", slog.Any("err", err))
	}
	write(w, http.StatusInternalServerError, Envelope{Error: "internal server error", Code: "internal"})
}

func reqContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

func write(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}
