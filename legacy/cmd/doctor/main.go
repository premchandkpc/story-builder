package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

type Check struct {
	Name     string
	Check    func() error
	Optional bool
}

func checkURL(url string) func() error {
	return func() error {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}
}

func checkPort(addr string) func() error {
	return func() error {
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}
}

func checkEnv(key string) func() error {
	return func() error {
		if os.Getenv(key) == "" {
			return fmt.Errorf("%s not set", key)
		}
		return nil
	}
}

func checkOptionalEnv(key string) func() error {
	return func() error {
		return nil
	}
}

func main() {
	checks := []Check{
		{Name: "Go API", Check: checkURL("http://localhost:8080/api/v1/healthz")},
		{Name: "MongoDB", Check: checkPort("localhost:27017")},
		{Name: "Redis", Check: checkPort("localhost:6379"), Optional: true},
		{Name: "Anthropic Key", Check: checkEnv("ANTHROPIC_API_KEY"), Optional: true},
		{Name: "OpenCode", Check: checkURL("http://localhost:11434/api/tags"), Optional: true},
		{Name: "Java Analysis", Check: checkURL("http://localhost:8081/actuator/health"), Optional: true},
		{Name: "OTL Exporter", Check: checkOptionalEnv("OTEL_EXPORTER_OTLP_ENDPOINT")},
	}

	allOK := true
	fmt.Println("=== Story Builder Health Check ===")
	for _, c := range checks {
		start := time.Now()
		err := c.Check()
		elapsed := time.Since(start).Round(time.Millisecond)
		status := "OK"
		if err != nil {
			if c.Optional {
				status = "SKIP"
			} else {
				status = "FAIL"
				allOK = false
			}
		}
		fmt.Printf("  %-18s %s  (%s)\n", c.Name+":", status, elapsed)
		if err != nil && !c.Optional {
			fmt.Printf("    └─ %v\n", err)
		}
	}
	if !allOK {
		os.Exit(1)
	}
}
