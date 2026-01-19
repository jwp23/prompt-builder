package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitForModel_AlreadyLoaded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForModel_QuietMode(t *testing.T) {
	// Server that would fail if called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called in quiet mode")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, true, true) // quiet=true

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForModel_NonTTY(t *testing.T) {
	// Server that would fail if called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called in non-TTY mode")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, false) // tty=false

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
