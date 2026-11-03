.PHONY: run test build web dev

run:
	go run ./cmd/server

test:
	go test ./...

build:
	go build ./...

web:
	cd web && npm run dev

dev:
	@echo "Run 'make run' and 'make web' in separate terminals."
