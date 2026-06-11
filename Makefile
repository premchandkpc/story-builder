.PHONY: all dev run frontend db db/stop build sqlc clean

all: build

dev: db
	@echo "Starting Go server (background) + Vite frontend..."
	@mkdir -p .tmp
	@-lsof -ti :8080 | xargs -r kill -9 >/dev/null 2>&1 || true
	@-lsof -ti :5173 | xargs -r kill -9 >/dev/null 2>&1 || true
	@trap 'kill $$SERVER_PID $$FRONTEND_PID 2>/dev/null || true' EXIT; \
		(go run ./cmd/server/ > .tmp/server.log 2>&1 &) ; \
		SERVER_PID=$$!; \
		(cd web && npm run dev -- --host 0.0.0.0 > ../.tmp/frontend.log 2>&1 &) ; \
		FRONTEND_PID=$$!; \
		wait $$SERVER_PID $$FRONTEND_PID

run:
	go run ./cmd/server/

frontend:
	cd web && npm run dev

db:
	docker compose up -d

db/stop:
	docker compose down

build:
	go build ./...
	cd web && npm run build

sqlc:
	sqlc generate -f sqlc/sqlc.yaml

clean:
	rm -rf web/dist
