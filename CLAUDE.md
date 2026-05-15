# CLAUDE.md

Read and follow `AGENTS.md` in this repository root — it is the primary operating guide.

## Git rules (non-negotiable)

- Always use conventional commit messages: `type(scope): summary`
- Never skip hooks. Never use `--no-verify`. If a hook fails, fix the issue.
- Never add "Co-Authored-By" lines to commit messages
- Never add "Generated with Claude Code" or any AI attribution to commits or PR descriptions
- Write helpful, substantive PR descriptions about what was actually worked on

## Project quick reference

- **Language**: Go 1.26.3, module `github.com/strait-dev/cli`
- **Build**: `go build ./...` or `make build`
- **Test**: `go test ./...` or `make test`
- **Lint**: `golangci-lint run ./...` or `make lint`
- **Install**: `go install ./cmd/strait` or `make install`
- **All checks**: `make check` (vet + lint + test)

## Key conventions

- This is a standalone REST API client — no server dependencies (no pgx, no chi, no queue, no worker imports)
- CLI-own types in `internal/types/` match the REST API JSON contract
- Error wrapping with `%w` and context
- No emojis in code, comments, logs, docs, or commits
- TTY output goes to stderr (styled), machine output goes to stdout (JSON)
