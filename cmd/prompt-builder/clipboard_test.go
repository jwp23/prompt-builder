// clipboard_test.go
package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDetectClipboardCmd(t *testing.T) {
	// This test verifies the detection logic
	// Actual availability depends on system
	cmd := DetectClipboardCmd("")

	// Should return something or empty string
	// Can't assert exact value as it's system-dependent
	t.Logf("Detected clipboard command: %q", cmd)

	// If a command is returned, it should be executable
	if cmd != "" {
		parts := strings.Split(cmd, " ")
		_, err := exec.LookPath(parts[0])
		if err != nil {
			t.Errorf("Detected command %q but binary not found", parts[0])
		}
	}
}

func TestDetectClipboardCmd_Override(t *testing.T) {
	cmd := DetectClipboardCmd("custom-clipboard")
	if cmd != "custom-clipboard" {
		t.Errorf("DetectClipboardCmd with override = %q, want %q", cmd, "custom-clipboard")
	}
}
