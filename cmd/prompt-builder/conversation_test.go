// conversation_test.go
package main

import "testing"

func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation("You are helpful.")

	// Should start with system message
	if len(conv.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", conv.Messages[0].Role, "system")
	}

	conv.AddUserMessage("Hello")
	if len(conv.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[1].Role != "user" {
		t.Errorf("second message role = %q, want %q", conv.Messages[1].Role, "user")
	}

	conv.AddAssistantMessage("Hi there!")
	if len(conv.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[2].Role != "assistant" {
		t.Errorf("third message role = %q, want %q", conv.Messages[2].Role, "assistant")
	}
}
