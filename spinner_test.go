package main

import (
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading...")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if s.message != "Loading..." {
		t.Errorf("message = %q, want %q", s.message, "Loading...")
	}
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic
	s.Stop()
}

func TestSpinner_StopMultipleTimes(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic on multiple Stop calls
	s.Stop()
	s.Stop()
	s.Stop()
}

func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner("Loading")
	s.Start()
	// Give it a moment to run
	time.Sleep(50 * time.Millisecond)
	s.Stop()
	// Should complete without hanging
}

func TestNewSpinnerWithTTY_False(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	if s.tty {
		t.Error("expected tty to be false")
	}
}

func TestNewSpinnerWithTTY_True(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", true)
	if !s.tty {
		t.Error("expected tty to be true")
	}
}

func TestSpinner_StartNonTTY(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	s.Start() // Should be no-op, not start goroutine
	s.Stop()  // Should be safe
}
