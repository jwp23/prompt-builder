package main

import "testing"

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
