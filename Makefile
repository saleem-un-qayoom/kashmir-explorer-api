# Kashmir Explorer API — dev shortcuts.
#
# Two ways to run Postgres locally:
#   1) `make db-start` — spins up a Docker container with PostGIS + pgvector
#      preinstalled (recommended; isolated from any system Postgres).
#   2) Bring your own Postgres on 5432 and run `make db-create` to add the
#      kashmir database + extensions.
#
# Both flows converge on `make migrate-up`.

.PHONY: run dev tidy sqlc test fmt swagger \
        migrate-up migrate-down migrate-status migrate-reset \
        db-start db-stop db-create db-init db-check

# Override by exporting DATABASE_URL or passing it on the command line.
DATABASE_URL ?= postgres://saleemunqayoom@localhost:5432/kashmir?sslmode=disable

# Connection URL pointing at the *server* (no specific db) for bootstrap.
SUPERUSER_URL ?= postgres://saleemunqayoom@localhost:5432/postgres?sslmode=disable

run:
	go run ./cmd/server

dev:
	@if command -v air >/dev/null 2>&1; then \
	  echo "  live-reload via air"; \
	  air -c .air.toml; \
	else \
	  echo "  air not installed — falling back to go run (install with: go install github.com/air-verse/air@latest)"; \
	  go run ./cmd/server; \
	fi

tidy:
	go mod tidy

# ── Migrations ───────────────────────────────────────────────

migrate-up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir db/migrations postgres "$(DATABASE_URL)" status

migrate-reset:
	goose -dir db/migrations postgres "$(DATABASE_URL)" reset

sqlc:
	sqlc generate

# Regenerate the OpenAPI/Swagger docs from handler annotations into docs/.
# Install the CLI once: go install github.com/swaggo/swag/cmd/swag@latest
swagger:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

test:
	go test ./... -race -cover

fmt:
	gofmt -s -w .

# ── Database lifecycle ───────────────────────────────────────

# Sanity-check: is Postgres reachable, and on what flavour?
db-check:
	@psql "$(SUPERUSER_URL)" -tAc "SELECT 'reachable as ' || current_user || ' on ' || version();" \
	  || { echo "❌ Cannot reach Postgres at $(SUPERUSER_URL)"; echo "   → Either run 'make db-start' or start your system Postgres."; exit 1; }

# Create the kashmir DB + extensions on *whatever* Postgres is reachable.
# Safe to run repeatedly; idempotent.
db-create:
	@echo "  Ensuring 'kashmir' database exists on $(SUPERUSER_URL)…"
	@psql "$(SUPERUSER_URL)" -tAc "SELECT 1 FROM pg_database WHERE datname='kashmir'" | grep -q 1 \
	  || psql "$(SUPERUSER_URL)" -c "CREATE DATABASE kashmir;"
	@echo "  Installing PostGIS + pgvector + pg_trgm + uuid-ossp…"
	@psql "$(DATABASE_URL)" -c "CREATE EXTENSION IF NOT EXISTS postgis;" \
	                       -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;" \
	                       -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" \
	                       -c "CREATE EXTENSION IF NOT EXISTS vector;" \
	  || { echo "❌ Extension install failed."; \
	       echo "   System Postgres (Homebrew/Postgres.app) usually doesn't ship PostGIS + pgvector."; \
	       echo "   → Easiest fix: 'make db-stop && make db-start' to use the Docker image which has them preinstalled."; \
	       exit 1; }
	@echo "✓ kashmir DB ready."

# One-shot: ensure DB exists and is migrated.
db-init: db-create migrate-up
	@echo "✓ DB initialised and migrated."

# Spin up the official pgvector image with PostGIS + pgvector preinstalled.
# If a container or system Postgres is already on :5432 this will fail; we
# detect that and tell the user how to recover.
db-start:
	@docker ps -a --format '{{.Names}}' | grep -q '^kashmir-pg$$' \
	  && docker start kashmir-pg > /dev/null \
	  || docker run -d --name kashmir-pg \
	      -e POSTGRES_PASSWORD=postgres \
	      -e POSTGRES_DB=kashmir \
	      -p 5432:5432 \
	      pgvector/pgvector:pg16
	@echo "  Waiting for Postgres to accept connections…"
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
	  docker exec kashmir-pg pg_isready -U postgres -q && break; \
	  sleep 1; \
	done
	@docker exec kashmir-pg psql -U postgres -d kashmir \
	  -c "CREATE EXTENSION IF NOT EXISTS postgis;" \
	  -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;" \
	  -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" \
	  -c "CREATE EXTENSION IF NOT EXISTS vector;" > /dev/null
	@echo "✓ Postgres ready on localhost:5432  (db=kashmir, user=postgres, pwd=postgres)"
	@echo "   Run 'make migrate-up' next."

db-stop:
	@docker rm -f kashmir-pg 2>/dev/null || true
	@echo "✓ kashmir-pg container removed."
