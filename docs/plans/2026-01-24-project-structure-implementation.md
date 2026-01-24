# Project Structure Reorganization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize the project to idiomatic Go layout with all source in `cmd/prompt-builder/`.

**Architecture:** Move all Go files to `cmd/prompt-builder/`, update build commands in README, simplify .gitignore for `go install` workflow.

**Tech Stack:** Go, git

---

## Task 1: Create Directory Structure and Move Files

**Files:**
- Create: `cmd/prompt-builder/` (directory)
- Move: All 16 `.go` files from root to `cmd/prompt-builder/`

**Step 1: Create the cmd directory structure**

Run:
```bash
mkdir -p cmd/prompt-builder
```

Expected: Directory created, no output.

**Step 2: Move all Go source files**

Run:
```bash
git mv main.go cmd/prompt-builder/
git mv config.go config_test.go cmd/prompt-builder/
git mv ollama.go ollama_test.go cmd/prompt-builder/
git mv conversation.go conversation_test.go cmd/prompt-builder/
git mv commands.go commands_test.go cmd/prompt-builder/
git mv clipboard.go clipboard_test.go cmd/prompt-builder/
git mv detect.go detect_test.go cmd/prompt-builder/
git mv spinner.go spinner_test.go cmd/prompt-builder/
git mv integration_test.go cmd/prompt-builder/
```

Expected: Files moved, staged for commit.

**Step 3: Verify files moved correctly**

Run:
```bash
ls cmd/prompt-builder/
```

Expected: All 16 .go files listed:
```
clipboard.go       commands.go       config.go       conversation.go
clipboard_test.go  commands_test.go  config_test.go  conversation_test.go
detect.go          integration_test.go  main.go      ollama.go
detect_test.go     ollama_test.go    spinner.go     spinner_test.go
```

**Step 4: Run tests to verify nothing broke**

Run:
```bash
go test ./cmd/prompt-builder
```

Expected: `ok  github.com/jwp23/prompt-builder/cmd/prompt-builder`

**Step 5: Commit**

```bash
git commit -m "refactor: move Go source files to cmd/prompt-builder/"
```

---

## Task 2: Remove Binary and Update .gitignore

**Files:**
- Delete: `prompt-builder` (binary in root)
- Modify: `.gitignore`

**Step 1: Remove the compiled binary**

Run:
```bash
rm -f prompt-builder
```

Expected: Binary deleted, no output.

**Step 2: Update .gitignore**

Replace contents of `.gitignore` with:

```
# Test binaries
*.test

# Go workspace
go.work
go.work.sum
```

**Step 3: Verify .gitignore is correct**

Run:
```bash
cat .gitignore
```

Expected: Shows the new simplified content (no `prompt-builder` line).

**Step 4: Stage and commit**

```bash
git add .gitignore
git commit -m "chore: simplify .gitignore for go install workflow"
```

---

## Task 3: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Update the Quick Start section**

Find the "2. Build" section and replace:

OLD:
```markdown
**2. Build:**

```bash
git clone https://github.com/mordant23/prompt-builder.git
cd prompt-builder
go build -o prompt-builder .
```
```

NEW:
```markdown
**2. Install:**

```bash
git clone https://github.com/jwp23/prompt-builder.git
cd prompt-builder
go install ./cmd/prompt-builder
```
```

**Step 2: Update the "4. Run" section**

OLD:
```markdown
**4. Run:**

```bash
./prompt-builder "a prompt for writing technical docs"
```
```

NEW:
```markdown
**4. Run:**

```bash
prompt-builder "a prompt for writing technical docs"
```
```

**Step 3: Update examples section**

Replace all `./prompt-builder` with `prompt-builder` in the Examples section.

OLD examples use: `./prompt-builder`
NEW examples use: `prompt-builder`

**Step 4: Add Project Structure section before Requirements**

Insert before the "## Requirements" section:

```markdown
## Project Structure

```
prompt-builder/
├── cmd/prompt-builder/   # CLI source code
├── docs/                 # Documentation
├── go.mod
└── README.md
```

## Development

```bash
# Install locally
go install ./cmd/prompt-builder

# Run tests
go test ./cmd/prompt-builder

# Run tests with coverage
go test -cover ./cmd/prompt-builder
```

```

**Step 5: Verify README renders correctly**

Run:
```bash
head -50 README.md
```

Expected: Shows updated installation instructions with `go install`.

**Step 6: Commit**

```bash
git add README.md
git commit -m "docs: update README for new project structure"
```

---

## Task 4: Final Verification

**Step 1: Verify go install works**

Run:
```bash
go install ./cmd/prompt-builder
```

Expected: No errors, binary installed to `$GOBIN`.

**Step 2: Verify binary runs**

Run:
```bash
prompt-builder --version
```

Expected: `prompt-builder dev` (or similar version output).

**Step 3: Verify all tests pass**

Run:
```bash
go test ./cmd/prompt-builder -v
```

Expected: All tests PASS.

**Step 4: Verify root directory is clean**

Run:
```bash
ls -la | grep -v "^\." | grep -v "^total"
```

Expected: Only these items in root:
- `cmd/`
- `docs/`
- `go.mod`
- `go.sum`
- `README.md`

**Step 5: Final commit message for any cleanup**

If any cleanup needed:
```bash
git add -A
git commit -m "chore: final cleanup for project restructure"
```

---

## Summary

After completing all tasks:
- All Go code lives in `cmd/prompt-builder/`
- Root contains only: `cmd/`, `docs/`, `go.mod`, `go.sum`, `README.md`, `.gitignore`
- Build with: `go install ./cmd/prompt-builder`
- Test with: `go test ./cmd/prompt-builder`
- No local binary to manage
