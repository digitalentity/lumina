# AGENTS.md

This file provides guidance to AI Agents when working with code in this repository.

## What this is

Lumina is a standalone Go CLI for the academic-writing pipeline: Mermaid
diagrams, citation checking, bibliography pruning/formatting, prose linting,
word-count enforcement, and PDF/DOCX/TeX/ZIP builds — all driven from a
manuscript directory (no fixed repo layout required). See `README.md` for the
full user-facing command reference and config schema; don't duplicate it here.

## Commands

```sh
make build     # -> _build/lumina
make test      # go test ./...
make vet       # go vet ./...
make install   # -> $HOME/.local/bin/lumina
make image     # build lumina-tools:latest (Docker runner's tool image)
```

Single package/test: `go test ./internal/citations/... -run TestName -v`

## Architecture

- `main.go` → `cmd.Execute()`. Cobra command tree: `cmd/root.go` wires
  subcommand groups `cmd/build`, `cmd/lit`, `cmd/text`, plus
  top-level `cmd/init.go`, `cmd/clean.go`.
- Every command (except `init`) loads the manuscript root via
  `internal/manuscript`, which requires `manuscript.md` to be present and
  resolves all other paths relative to it.
- `internal/config`: parses `metadata.yaml` (pandoc-standard keys forwarded
  verbatim, `wordlimit` stripped, `acronyms` reshaped for `pandoc-acro`) and
  `lumina.yaml` (tool/runner config).
- `internal/runner`: **all external tool invocations** (`pandoc`, `mmdc`,
  `vale`, `prettier`, TeX engine, `zip`) go through this
  abstraction — never shell out directly from command code. Two
  implementations: `host.go` (runs on `$PATH`) and `docker.go` (runs each
  call as `docker run --rm -v <root>:/workspace ... <tools-image> <tool>
  <args>`), selected by `runner: host|docker` in `lumina.yaml`.
- `internal/preprocess`: renders Mermaid blocks to PNG (cached by
  `SHA-256(block)[:16]` under `.lumina/figures/`), stages `figures/`,
  `references.bib`, `metadata.yaml` into `.lumina/` ahead of a build.
- `internal/pandoc`: builds pandoc invocations for pdf/docx/tex targets.
  `internal/bibtex`, `internal/citations`: bibliography parsing and
  in-manuscript citation-key extraction, shared by `lit check`/`lit
  prune`/`lit fmt`.
- `internal/scaffold` (+ `scaffold/templates`): backs `lumina init`.
- `internal/logx`: shared colorized logger used across all commands.

## Design docs

Features go through spec-driven design under `spec/<NNN_name>/spec.md`
(template: `spec/TEMPLATE.md`, index: `spec/README.md`). Check there for
the "why" behind a feature before re-deriving it from code.

## Conventions

- Command code never calls external tools directly — go through
  `internal.Runner` so both host and Docker execution modes stay correct.
- Build subcommands (`build pdf`/`docx`/`tex`/`zip`) re-run `preprocess`
  first if stale, and accept `--force`/`-f` to bypass caches
  (Mermaid PNGs).
- Mutating commands that touch user files (`lit prune`) default to
  dry-run and require `--no-dry-run` (+ confirmation unless `--yes`/`-y`).
