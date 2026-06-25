package elasticsearch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	es "github.com/elastic/go-elasticsearch/v8"
)

type Client struct {
	raw *es.TypedClient
}

func Connect(addrs []string) (*Client, error) {
	cfg := es.Config{
		Addresses: addrs,
		Transport: &traceTransport{},
	}
	raw, err := es.NewTypedClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch connect: %w", err)
	}
	client := &Client{raw: raw}
	if err := client.ping(); err != nil {
		return nil, fmt.Errorf("elasticsearch ping: %w", err)
	}
	return client, nil
}

func ConnectWithPassword(addrs []string, password string) (*Client, error) {
	cfg := es.Config{
		Addresses: addrs,
		Username:  "elastic",
		Password:  password,
		Transport: &traceTransport{},
	}
	raw, err := es.NewTypedClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch connect: %w", err)
	}
	client := &Client{raw: raw}
	if err := client.ping(); err != nil {
		return nil, fmt.Errorf("elasticsearch ping: %w", err)
	}
	return client, nil
}

func (c *Client) ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.raw.Ping().Do(ctx)
	return err
}

func (c *Client) Raw() *es.TypedClient {
	return c.raw
}

type traceTransport struct{}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := http.DefaultTransport.RoundTrip(req)
	duration := time.Since(start)
	if err != nil {
		slog.Debug("es request failed", "method", req.Method, "url", req.URL.Path, "duration", duration, "error", err)
	} else {
		slog.Debug("es request", "method", req.Method, "url", req.URL.Path, "status", resp.StatusCode, "duration", duration)
	}
	return resp, err
}
