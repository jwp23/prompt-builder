# Project Structure Reorganization

Reorganize the project to follow idiomatic Go project layout for CLI applications.

## Goal

Clean up the root directory by moving all Go source files into `cmd/prompt-builder/`. This follows standard Go conventions and makes the project easier to understand at a glance.

## Current State

All 16 Go files (8 source + 8 test files) sit in the root directory alongside go.mod, README, etc. The compiled binary also lives in root.

## Target Structure

```
prompt-builder/
├── cmd/
│   └── prompt-builder/
│       ├── main.go
│       ├── config.go
│       ├── config_test.go
│       ├── ollama.go
│       ├── ollama_test.go
│       ├── conversation.go
│       ├── conversation_test.go
│       ├── commands.go
│       ├── commands_test.go
│       ├── clipboard.go
│       ├── clipboard_test.go
│       ├── detect.go
│       ├── detect_test.go
│       ├── spinner.go
│       ├── spinner_test.go
│       └── integration_test.go
├── docs/
│   └── plans/
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

## Changes

### 1. Create directory and move files

Create `cmd/prompt-builder/` and move all 16 .go files there.

### 2. Delete binary from root

Remove the compiled `prompt-builder` binary. Going forward, use `go install` instead of local builds.

### 3. Update .gitignore

Remove binary-specific ignore. New content:

```
# Test binaries
*.test

# Go workspace
go.work
go.work.sum
```

### 4. Update README.md

Update installation and development instructions:

- Installation: `go install github.com/jwp23/prompt-builder/cmd/prompt-builder@latest`
- Development: `go install ./cmd/prompt-builder`
- Testing: `go test ./cmd/prompt-builder`
- Add project structure section

## Build Commands

After reorganization:

```bash
# Install to $GOBIN (recommended)
go install ./cmd/prompt-builder

# Run tests
go test ./cmd/prompt-builder

# Run tests with coverage
go test -cover ./cmd/prompt-builder
```

## Verification

After implementation:
1. `go test ./cmd/prompt-builder` passes
2. `go install ./cmd/prompt-builder` succeeds
3. `prompt-builder --version` works
