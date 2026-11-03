FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /story-builder ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /story-builder /story-builder
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/story-builder"]
