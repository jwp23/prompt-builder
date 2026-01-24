// conversation.go
package main

type Conversation struct {
	Messages []Message
}

func NewConversation(systemPrompt string) *Conversation {
	return &Conversation{
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
		},
	}
}

func (c *Conversation) AddUserMessage(content string) {
	c.Messages = append(c.Messages, Message{Role: "user", Content: content})
}

func (c *Conversation) AddAssistantMessage(content string) {
	c.Messages = append(c.Messages, Message{Role: "assistant", Content: content})
}
