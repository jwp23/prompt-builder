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
