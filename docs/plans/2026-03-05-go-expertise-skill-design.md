# go-expertise Skill Design

Personal skill for Claude to write idiomatic Go and teach Go patterns while coding.

## Scope

**Triggers:** When writing, reviewing, or modifying Go code.

**Covers:** Idiomatic Go patterns, project structure, tooling, teaching annotations.

**Does not cover:** Framework-specific guidance (Bubbletea, Gin, etc.).

## Core Patterns

- **Error handling:** Return errors, wrap with `%w`, custom error types, never ignore
- **Naming:** Short names in small scopes, MixedCaps, `-er` interfaces, short package names
- **Interfaces:** Accept interfaces return structs, small interfaces, embed for composition
- **Concurrency:** Channels over shared memory, `context.Context` for cancellation
- **Project structure:** `cmd/`, `internal/`, standard module layout
- **Tooling:** `go mod`, `go test`, `go vet`, `golangci-lint`, table-driven tests, Makefile

## Teaching Behavior

1. **Inline annotations** - Brief `// Why:` comments on non-obvious Go choices
2. **Code review mode** - Flag non-idiomatic patterns with fix + one-line explanation
3. **Fade-out rule** - Don't explain basic patterns after they've been seen multiple times

## Skill Structure

Single `SKILL.md` at `~/.claude/skills/go-expertise/SKILL.md`. ~300-400 words. Sections:

1. Frontmatter (name, description)
2. Overview
3. Idiomatic Patterns (quick reference tables)
4. Tooling
5. Teaching Rules
6. Common Mistakes table
