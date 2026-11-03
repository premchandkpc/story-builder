.PHONY: all dev dev-hot run frontend db db/stop build lint test test-race test-cover fmt doctor clean

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

dev-hot: db
	@echo "Starting with hot-reload (air)..."
	@which air > /dev/null 2>&1 || (echo "Installing air..."; go install github.com/air-verse/air@latest)
	@air -c .air.toml

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

fmt:
	go fmt ./...

test:
	go test ./... -count=1

test-race:
	go test ./... -count=1 -race -timeout 120s

test-cover:
	go test ./... -count=1 -coverprofile=tmp/cover.out
	go tool cover -html=tmp/cover.out -o tmp/cover.html
	@echo "Coverage report: tmp/cover.html"

doctor:
	@echo "=== Story Builder Health Check ==="
	@printf "Go API       "; curl -sf http://localhost:8080/api/v1/healthz > /dev/null && echo "✅" || echo "❌"
	@printf "MongoDB      "; nc -z localhost 27017 2>/dev/null && echo "✅" || echo "❌"
	@printf "Redis        "; nc -z localhost 6379 2>/dev/null && echo "✅" || echo "❌"
	@printf "Anthropic Key"; test -n "$$ANTHROPIC_API_KEY" && echo "✅" || echo "⚠  (optional)"
	@printf "OpenCode     "; curl -sf http://localhost:11434/api/tags > /dev/null 2>&1 && echo "✅" || echo "⚠  (optional)"
	@printf "Java Service "; curl -sf http://localhost:8081/actuator/health > /dev/null 2>&1 && echo "✅" || echo "⚠  (optional)"
	@echo ""

clean:
	rm -rf web/dist tmp/
