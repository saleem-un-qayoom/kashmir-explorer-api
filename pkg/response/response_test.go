package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env
}

func TestOK(t *testing.T) {
	rec := httptest.NewRecorder()
	OK(rec, map[string]string{"hello": "world"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q", ct)
	}
	env := decode(t, rec)
	data, ok := env.Data.(map[string]any)
	if !ok || data["hello"] != "world" {
		t.Errorf("Data = %v, want {hello: world}", env.Data)
	}
	if env.Error != "" {
		t.Errorf("Error = %q, want empty", env.Error)
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestErrorHelpers(t *testing.T) {
	cases := []struct {
		name    string
		fn      func(http.ResponseWriter, string)
		msg     string
		status  int
		code    string
		wantMsg string
	}{
		{"bad_request", BadRequest, "nope", http.StatusBadRequest, "bad_request", "nope"},
		{"unauthorized_default", Unauthorized, "", http.StatusUnauthorized, "unauthorized", "authentication required"},
		{"forbidden_default", Forbidden, "", http.StatusForbidden, "forbidden", "forbidden"},
		{"not_found_default", NotFound, "", http.StatusNotFound, "not_found", "not found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.fn(rec, c.msg)
			if rec.Code != c.status {
				t.Errorf("status = %d, want %d", rec.Code, c.status)
			}
			env := decode(t, rec)
			if env.Code != c.code {
				t.Errorf("code = %q, want %q", env.Code, c.code)
			}
			if env.Error != c.wantMsg {
				t.Errorf("error = %q, want %q", env.Error, c.wantMsg)
			}
		})
	}
}

func TestFromError(t *testing.T) {
	t.Run("no rows → 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		FromError(rec, httptest.NewRequest(http.MethodGet, "/", nil), pgx.ErrNoRows, "destination not found")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		if env := decode(t, rec); env.Error != "destination not found" {
			t.Errorf("error = %q", env.Error)
		}
	})

	t.Run("unique violation → 409", func(t *testing.T) {
		rec := httptest.NewRecorder()
		FromError(rec, httptest.NewRequest(http.MethodGet, "/", nil), &pgconn.PgError{Code: "23505"}, "")
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", rec.Code)
		}
		if env := decode(t, rec); env.Code != "conflict" {
			t.Errorf("code = %q, want conflict", env.Code)
		}
	})

	t.Run("other → 500 generic", func(t *testing.T) {
		rec := httptest.NewRecorder()
		FromError(rec, httptest.NewRequest(http.MethodGet, "/", nil), errSentinel, "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		// The raw error text must never leak to the client.
		if env := decode(t, rec); env.Error == errSentinel.Error() {
			t.Error("internal error text leaked to client")
		}
	})
}

type sentinel struct{}

func (sentinel) Error() string { return "secret driver detail" }

var errSentinel = sentinel{}
