// detect_test.go
package main

import "testing"

func TestExtractLastCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "single code block",
			input: `Here is your prompt:
` + "```" + `
# Role
You are an expert.
` + "```" + `
`,
			want: `# Role
You are an expert.
`,
		},
		{
			name: "multiple code blocks - returns last",
			input: `Example:
` + "```" + `
first block
` + "```" + `

Here is the final:
` + "```" + `
second block
` + "```" + `
`,
			want: `second block
`,
		},
		{
			name:  "no code block",
			input: "Just plain text",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLastCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("ExtractLastCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsComplete(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "code block without question - complete",
			input: "Here is your prompt:\n```\ncontent\n```\n",
			want:  true,
		},
		{
			name:  "code block with trailing question - not complete",
			input: "Here is a draft:\n```\ncontent\n```\nDoes this look right?",
			want:  false,
		},
		{
			name:  "question only - not complete",
			input: "What is your target audience?",
			want:  false,
		},
		{
			name:  "no code block no question - not complete",
			input: "Let me think about that.",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsComplete(tt.input)
			if got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}
