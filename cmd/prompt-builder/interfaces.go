// interfaces.go
package main

import "io"

// Deps holds injectable dependencies for the app.
type Deps struct {
	Client    OllamaChatter
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Clipboard ClipboardWriter
	IsTTY     func() bool
}
