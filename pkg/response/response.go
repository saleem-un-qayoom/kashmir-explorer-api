// Package response — consistent JSON envelope helpers.
package response

import (
	"encoding/json"
	"net/http"
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

func Internal(w http.ResponseWriter, err error) {
	msg := "internal server error"
	if err != nil {
		msg = err.Error()
	}
	write(w, http.StatusInternalServerError, Envelope{Error: msg, Code: "internal"})
}

func write(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}
