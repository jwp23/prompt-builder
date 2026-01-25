package main

import (
	"context"
	"testing"
)

func TestOllamaClient_ImplementsOllamaChatter(t *testing.T) {
	var _ OllamaChatter = (*OllamaClient)(nil)
}

func TestRunWithDeps_Exists(t *testing.T) {
	// Just verify the function signature exists
	var _ func(context.Context, *CLI, *Deps) error = runWithDeps
}
