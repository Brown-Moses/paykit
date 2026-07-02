.PHONY: up down ps migrate ping-db run build tidy

# ── Docker ────────────────────────────────────────────────────────────────────────
up:
	docker-compose up -d

down:
	docker-compose down

ps:
	docker-compose ps

# ── Database ──────────────────────────────────────────────────────────────────────
migrate:
	@# Run migration in a safe/idempotent way (multiple runs should not error)
	docker exec -i paykit_postgres psql -U paykit -d paykit -v ON_ERROR_STOP=1 < internal/storage/migrate.sql


ping-db:
	docker exec paykit_postgres pg_isready -U paykit

# ── App ───────────────────────────────────────────────────────────────────────────
run:
	go run cmd/paykit/main.go

build:
	go build ./...

tidy:
	go mod tidy

# ── Backup ────────────────────────────────────────────────────────────────────────
db-backup:
	docker exec paykit_postgres pg_dump -U paykit -d paykit > backups/paykit_$(shell date +%Y%m%d_%H%M%S).sql

db-restore FILE=:
	cat $(FILE) | docker exec -i paykit_postgres psql -U paykit -d paykit

swagger:
	swag init -g cmd/paykit/main.go --output docs

health:
	curl -s http://localhost:8080/health | jq

metrics:
	curl -s http://localhost:8080/metrics | jq