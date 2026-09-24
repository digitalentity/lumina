# AGENTS.md

This file provides guidance to AI Agents when working with code in this repository.

## What this is

Lumina is a standalone Go CLI for the academic-writing pipeline: Mermaid
diagrams, citation checking, bibliography pruning/formatting, prose linting,
word-count enforcement, and PDF/DOCX/TeX/ZIP builds — driven from a project root
containing one or more manuscript targets under `src/<target>/`. See `README.md` for the
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
  subcommands `cmd/build`, `cmd/lit`, `cmd/text`, plus
  top-level `cmd/log.go`, `cmd/init.go`, `cmd/clean.go`.
- Commands operate on targets: `lumina build <target> [--pdf|--docx|--tex|...]`,
  `lumina log <target> [--terminal|--stat|--pdf]`,
  `lumina lit check <target>`, `lumina text words <target>`.
- `internal/changelog`: extracts pure Go Git history (`go-git/v5`) for `src/<target>/manuscript.md`
  and renders paragraph-level colored diffs with `<del>` / `<ins>` in HTML, terminal ANSI, and PDF.
- `internal/manuscript`: loads target via `manuscript.Load(target)`. Discovers project
  root (searching for `lumina.yaml` or `src/`), requires `src/<target>/manuscript.md`,
  resolves project-level assets (`csl/`, `templates/<template>/`, `build/`),
  and computes output file stem (`output` in `metadata.yaml` or target name).
- `internal/config`: parses `lumina.yaml` (project root) and target `metadata.yaml`
  (pandoc-standard keys forwarded verbatim, `wordlimit`, `output`, and `template`
  stripped; `acronyms` reshaped for `pandoc-acro`).
- `internal/runner`: **all external tool invocations** (`pandoc`, `mmdc`,
  `vale`, `prettier`, TeX engine, `zip`) go through this
  abstraction — never shell out directly from command code. Two
  implementations: `host.go` (runs on `$PATH`) and `docker.go` (runs each
  call as `docker run --rm -v <root>:/workspace ... <tools-image> <tool>
  <args>`), selected by `runner: host|docker` in `lumina.yaml`.
- `internal/preprocess`: cleans temporary staging directory `.lumina/build/` before
  each build. Renders Mermaid blocks to PNG (cached persistently by `SHA-256(block)[:16]`
  under `.lumina/figures/`), stages `src/<target>/figures/`, `references.bib`,
  target `metadata.yaml`, resolved CSL file, and template styles into `.lumina/build/`.
- `internal/pandoc`: builds pandoc invocations for pdf/docx/tex targets.
- `internal/bibtex`, `internal/citations`: bibliography parsing and
  in-manuscript citation-key extraction, shared by `lit check`/`lit
  prune`/`lit fmt`.
- `internal/scaffold` (+ `scaffold/templates`): backs `lumina init <target>`. Scaffolds
  project root configuration (`lumina.yaml`, `csl/`, `templates/default/`, `.vale.ini`,
  `.gitignore`) if absent, plus target files in `src/<target>/`.
- `internal/logx`: shared colorized logger used across all commands.

## Design docs

Features go through spec-driven design under `spec/<NNN_name>/spec.md`
(template: `spec/TEMPLATE.md`, index: `spec/README.md`). Check there for
the "why" behind a feature before re-deriving it from code.

## Conventions

- Command code never calls external tools directly — go through
  `internal.Runner` so both host and Docker execution modes stay correct.
- Build target command (`build <target>`) re-runs `preprocess` first if stale,
  outputs to `build/<stem>.<ext>`, and accepts `--force`/`-f` to bypass caches
  (Mermaid PNGs).
- Mutating commands that touch user files (`lit prune`) default to
  dry-run and require `--no-dry-run` (+ confirmation unless `--yes`/`-y`).
