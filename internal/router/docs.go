package router

import (
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	// Blank import wires the generated OpenAPI spec into the swag registry so
	// httpSwagger can serve it. Regenerate with `make swagger` (or `go generate`).
	_ "github.com/kashmir-explorer/api/docs"
)

// registerDocs serves the Swagger UI and the generated OpenAPI document at
// /swagger/* (e.g. /swagger/index.html, /swagger/doc.json).
func registerDocs(r chi.Router) {
	r.Get("/swagger/*", httpSwagger.WrapHandler)
}
