# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go CLI project for video cleanup.

- `cmd/viso/`: CLI entrypoint (`main.go`) and command-level tests.
- `internal/scanner/`: file discovery and metadata scan orchestration.
- `internal/video/`: ffprobe/ffmpeg integration and video feature extraction.
- `internal/rules/`: cleanup rule engine (`duplicate`, `duration`, `resolution`).
- `internal/processor/`: safe move-to-trash behavior.
- `docs/`: project notes and AI change logs.

Keep business logic inside `internal/*`; keep `cmd/viso` thin.

## Build, Test, and Development Commands
- `go mod tidy`: sync and clean dependencies.
- `go test ./...`: run all tests.
- `go build -o viso ./cmd/viso/main.go`: build local binary.
- `./viso scan [dir]`: run scan on a target directory.
- Example: `./viso scan ~/Videos -s 7 -d 10s -W 640 -H 360`

Run `go test ./...` before every commit.

## Coding Style & Naming Conventions
- Follow standard Go formatting (`gofmt`) and idioms.
- Use tabs/Go default formatting; do not manually align with spaces.
- Package names are short, lowercase (`scanner`, `rules`).
- Exported identifiers use `CamelCase`; unexported use `camelCase`.
- Prefer descriptive names (`minDuration`, `PartialHash`) over abbreviations.

## Testing Guidelines
- Use Go’s built-in `testing` package.
- Place tests next to code as `*_test.go`.
- Name tests as behavior statements, e.g. `TestRun_ScanError_ReturnsCode1`.
- Cover argument parsing, error paths, and rule selection behavior.
- Run focused tests during development (example: `go test ./cmd/viso`), then full suite.

## Commit & Pull Request Guidelines
- Follow Conventional Commit style seen in history: `feat: ...`, `fix: ...`, `docs: ...`.
- Keep commits scoped to one intent (feature, bugfix, or docs update).
- PRs should include:
  - summary of behavior changes,
  - test evidence (command + result),
  - impacted paths (e.g., `cmd/viso/main.go`, `internal/rules/*`),
  - screenshots only when terminal UX/output meaningfully changes.

## Security & Configuration Tips
- `ffmpeg` and `ffprobe` must be available in `PATH`.
- Never hardcode absolute local paths in code or docs.
- Keep `.viso-trash` excluded from scans; avoid destructive delete commands in automation.
