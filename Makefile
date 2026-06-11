.PHONY: all dev run frontend db db/stop build sqlc clean

all: build

dev: db
	@echo "Starting Go server (background) + Vite frontend..."
	@trap 'kill 0' EXIT; \
		go run ./cmd/server/ & \
		cd web && npm run dev & \
		wait

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
