package main

import "testing"

func TestAPIHTTPAddrUsesConfiguredValue(t *testing.T) {
	lookup := func(key string) (string, bool) {
		if key == "API_HTTP_ADDR" {
			return "127.0.0.1:8080", true
		}
		return "", false
	}
	if got := apiHTTPAddr(lookup); got != "127.0.0.1:8080" {
		t.Fatalf("apiHTTPAddr() = %q, want 127.0.0.1:8080", got)
	}
}

func TestAPIHTTPAddrFallsBackToPort8080(t *testing.T) {
	if got := apiHTTPAddr(func(string) (string, bool) { return "", false }); got != ":8080" {
		t.Fatalf("apiHTTPAddr() = %q, want :8080", got)
	}
}
