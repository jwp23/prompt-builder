// detect.go
package main

import (
	"strings"
)

func ExtractLastCodeBlock(text string) string {
	const marker = "```"

	lastStart := strings.LastIndex(text, marker)
	if lastStart == -1 {
		return ""
	}

	// Find the opening marker for this block
	beforeLast := text[:lastStart]
	openStart := strings.LastIndex(beforeLast, marker)
	if openStart == -1 {
		return ""
	}

	// Extract content between markers
	// Skip past the opening ``` and any language identifier on that line
	contentStart := openStart + len(marker)
	if idx := strings.Index(text[contentStart:lastStart], "\n"); idx != -1 {
		contentStart += idx + 1
	}

	return text[contentStart:lastStart]
}
