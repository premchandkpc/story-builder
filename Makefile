.PHONY: all dev run frontend db db/stop build lint test clean

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
	@echo "Waiting for MongoDB..."
	@for i in $$(seq 1 30); do \
		if docker compose exec -T mongo mongosh --quiet --eval 'db.runCommand("ping").ok' >/dev/null 2>&1; then \
			echo "MongoDB ready"; \
			break; \
		fi; \
		echo "Waiting... $$i"; \
		sleep 1; \
	done
	@echo "Waiting for Redis..."
	@for i in $$(seq 1 15); do \
		if docker compose exec -T redis redis-cli -a storybuilder ping >/dev/null 2>&1; then \
			echo "Redis ready"; \
			break; \
		fi; \
		echo "Waiting... $$i"; \
		sleep 1; \
	done

db/stop:
	docker compose down

build:
	go build ./...
	cd web && npm run build

lint:
	golangci-lint run

test:
	go test ./...

clean:
	rm -rf web/dist
