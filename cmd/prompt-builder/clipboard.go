// clipboard.go
package main

import (
	"os/exec"
	"strings"
)

func DetectClipboardCmd(override string) string {
	if override != "" {
		return override
	}

	candidates := []string{
		"wl-copy",
		"xclip -selection clipboard",
		"xsel --clipboard --input",
		"pbcopy",
	}

	for _, cmd := range candidates {
		parts := strings.Split(cmd, " ")
		if _, err := exec.LookPath(parts[0]); err == nil {
			return cmd
		}
	}

	return ""
}

func CopyToClipboard(text string, cmd string) error {
	if cmd == "" {
		return nil // No clipboard available, silently skip
	}

	parts := strings.Split(cmd, " ")
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdin = strings.NewReader(text)
	return c.Run()
}
