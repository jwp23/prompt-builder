package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestWaitForModel_PollsUntilLoaded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			fmt.Fprintln(w, `{"models":[]}`) // Not loaded yet
		} else {
			fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`) // Now loaded
		}
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}

func TestWaitForModel_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[]}`) // Never becomes loaded
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModelWithTimeout(client, false, true, 100*time.Millisecond)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err.Error() != "timeout waiting for model to load" {
		t.Errorf("unexpected error: %v", err)
	}
}
