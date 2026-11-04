package main

import "testing"

func TestNonLoopbackRequiresAuthentication(t *testing.T) {
	for _, address := range []string{":8080", "0.0.0.0:8080", "[::]:8080", "192.0.2.10:8080"} {
		if validateListener(address, "") == nil {
			t.Fatalf("unauthenticated listener allowed: %s", address)
		}
		if err := validateListener(address, "token"); err != nil {
			t.Fatal(err)
		}
	}
	for _, address := range []string{"127.0.0.1:8080", "[::1]:8080", "localhost:8080"} {
		if err := validateListener(address, ""); err != nil {
			t.Fatal(err)
		}
	}
}
