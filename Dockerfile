FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /build/server ./cmd/server/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/server .
# Note: migrations/ skipped — MongoDB is schemaless (ADR 0003)

EXPOSE 8080

ENTRYPOINT ["/app/server"]
