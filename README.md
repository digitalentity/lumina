# lumina

Lumina is a standalone CLI for the academic-writing pipeline: Mermaid
diagrams, citation checking, bibliography pruning/formatting, prose linting,
word-count enforcement, and PDF/DOCX/TeX/ZIP builds — all driven from a
single manuscript directory, no Makefile or fixed repo layout required.

## Install

```sh
make build      # compiles ./_build/lumina
make install    # copies it to $HOME/.local/bin/lumina
```

External tools (`pandoc`, `pandoc-crossref`, `pandoc-acro`, `mmdc`, `vale`,
`prettier`, a TeX engine, `zip`) are not bundled in the binary. Either
install them on the host, or build the Docker tools image and set
`runner: docker` in `lumina.yaml` (see [Runner](#runner-host-vs-docker)):

```sh
make image      # builds lumina-tools:latest
```

## Quick start

```sh
mkdir my-paper && cd my-paper
lumina init            # scaffold manuscript.md, metadata.yaml, etc.
lumina build pdf        # -> _build/manuscript.pdf
```

## Directory layout

Lumina is invoked from within a manuscript directory; every path is
resolved relative to the current working directory.

| Path               | Purpose |
|---------------------|---------|
| `manuscript.md`     | The manuscript source. Pure prose, no YAML frontmatter. Only accepted filename — lumina fails fast if absent. |
| `metadata.yaml`      | All manuscript/rendering metadata (see below). |
| `references.bib`    | BibTeX database. Single file, consulted by `lit check`, `lit prune`, and the build. |
| `literature/`       | User-managed PDF library with same-stem `.bib` sidecars. Read-only to lumina. |
| `figures/`          | Static images/diagrams to include in the rendered paper. |
| `publish/template.tex` | Optional custom pandoc LaTeX template, used by `build pdf`/`build tex` if present. |
| `publish/reference.docx` | Optional custom Word reference doc, used by `build docx` if present. |
| `.vale.ini`          | Vale prose-linter config, discovered in the manuscript root. |
| `lumina.yaml`        | Tool configuration (see below). |
| `.lumina/`           | Intermediate state: preprocessed Markdown, rendered Mermaid PNGs, staged metadata/bib/figures. Not committed. |
| `_build/`            | Final artifacts: PDF, DOCX, TeX, ZIP. Not committed. |

`lumina init` scaffolds `manuscript.md`, `metadata.yaml`, `references.bib`,
`literature/`, `figures/`, `.vale.ini`, and `.gitignore`. It never
overwrites existing files.

## Commands

All commands are `lumina <group> <subcommand> [flags]`, run from within a
manuscript directory. Every command except `init` requires `manuscript.md`
to be present.

### Top-level

| Command | Description |
|---------|-------------|
| `lumina init` | Scaffold a new manuscript directory in the current directory. Safe to re-run; never overwrites. |
| `lumina clean` | Remove `.lumina/` and `_build/`. Leaves everything else untouched. |

### `lumina build` — compilation

| Command | Description |
|---------|-------------|
| `lumina build preprocess [--force/-f]` | Render Mermaid diagrams, stage `figures/`, `references.bib`, and `metadata.yaml` into `.lumina/`. `--force` re-renders all Mermaid PNGs, ignoring the cache. |
| `lumina build pdf [--pdf-engine ENGINE] [--force/-f]` | Build `_build/manuscript.pdf`. Re-runs `preprocess` first if stale. |
| `lumina build docx [--force/-f]` | Build `_build/manuscript.docx`. |
| `lumina build tex [--force/-f]` | Build `_build/manuscript.tex` (standalone LaTeX source — useful for Overleaf, or as input to `zip`). |
| `lumina build zip [--force/-f]` | Build `_build/manuscript.zip`: `.tex` + `references.bib` + `figures/`. Rebuilds the `.tex` first if it's stale (or `--force`). |
| `lumina build all [--force/-f]` | Citation check, then preprocess, then build every format listed in `lumina.yaml`'s `formats` (default: `pdf docx tex zip`). This is also what plain `lumina build` (no subcommand) runs. |
| `lumina build pub` | Pre-submission gate: citation check → Vale lint → word-limit check → TODO scan, all fail-fast. On success, builds PDF + ZIP and copies them to dated files: `_build/manuscript_<YYYY-MM-DD>.{pdf,zip}`. |

### `lumina text` — prose quality

| Command | Description |
|---------|-------------|
| `lumina text words` | Word count via `pandoc --to=plain`. Reports `count / limit` if `wordlimit` is set in `metadata.yaml`; over-limit is a warning here, not a failure (`build pub` is the enforcement gate). |
| `lumina text fmt` | `prettier --write manuscript.md`. |
| `lumina text lint` | Runs Vale against `manuscript.md`, using `.vale.ini` from the manuscript root. Runs `vale sync` first if `styles/` is absent. Non-zero exit on any Vale error. |

### `lumina lit` — literature & bibliography

| Command | Description |
|---------|-------------|
| `lumina lit check` | Verifies every `@key` cited in `manuscript.md` has a matching `references.bib` entry (fatal if not). Also reports non-fatal warnings: duplicate keys, duplicate DOIs, duplicate titles, missing required fields per entry type. |
| `lumina lit prune [--no-dry-run] [--yes/-y]` | Removes bibliography entries not cited in `manuscript.md`. **Dry-run by default** — reports what would be removed without touching the file. Pass `--no-dry-run` to actually rewrite `references.bib` (prompts for confirmation unless `--yes`). |
| `lumina lit fmt` | Formats `references.bib` in place: sorted entries, normalized whitespace/quoting. Does not add or remove entries. |

## Configuration

### `metadata.yaml`

Manuscript metadata, split into three groups:

- **Pandoc-standard keys**, forwarded to pandoc verbatim via
  `--metadata-file`: `title`, `author`, `date`, `bibliography`, `csl`,
  `numbersections`, `geometry`, `linestretch`, `toc`, `lof`, `lot`, etc.
- **Lumina-specific**, stripped before forwarding: `wordlimit` (integer
  word cap; `0` = unlimited, enforced by `build pub`).
- **Reshaped-and-forwarded**: `acronyms`. Written by the author as a flat
  `KEY: "definition"` map; lumina reshapes it into the `pandoc-acro`
  filter's schema (`KEY: {short: KEY, long: "definition"}`) before writing
  `.lumina/metadata.yaml`. Reference an acronym in the manuscript body with
  `+KEY` — `pandoc-acro` expands it to the long form + short form in
  parentheses on first use, short form only after that.

```yaml
title:          "Untitled Manuscript"
author:         "Author Name"
date:           "2026-07-08"
bibliography:   "references.bib"
csl:            "ieee.csl"
numbersections: true

wordlimit: 8000

acronyms:
  API: "Application Programming Interface"
  CLI: "Command Line Interface"
```

### `lumina.yaml`

Tool/environment configuration — not manuscript metadata. All keys optional.

```yaml
pdf-engine:  xelatex                # pandoc --pdf-engine value
formats:                            # formats built by "lumina build all"
  - pdf
  - docx
  - tex
  - zip
runner:      host                   # host | docker
tools-image: lumina-tools:latest    # used when runner: docker

ai:
  provider:         gemini              # gemini | openai
  model:            gemini-2.5-flash    # LLM model name
  base-url:         ""                  # optional custom endpoint base URL
  temperature:      0.2                 # temperature for checks
  search-method:    bm25                # bm25 | embeddings
  search-threshold: 0.0                 # minimum score/similarity filter
  embedding-model:  gemini-embedding-2  # embedding model name
```

### Runner: host vs. Docker

Lumina never calls external tools directly — every invocation goes through
a `Runner`. With `runner: host` (default), tools run via the host's `PATH`.
With `runner: docker`, each tool call becomes its own
`docker run --rm -v <manuscript-root>:/workspace -w /workspace <tools-image> <tool> <args...>`
against the image built by `make image` in this repo — no long-running
container, no host tool installation required.

## Mermaid diagrams

Any ` ```mermaid ` fenced code block in `manuscript.md` is rendered to a
PNG during `build preprocess`. Blocks are cached by
`SHA-256(block source)[:16]` under `.lumina/figures/mermaid-<hash>.png`;
identical diagrams across builds are not re-rendered. Pass `--force` to any
`build` subcommand to bypass the cache and re-render everything.

## Developing lumina itself

```sh
make build     # -> _build/lumina
make test      # go test ./...
make vet       # go vet ./...
make install   # -> $HOME/.local/bin/lumina
make image     # build lumina-tools:latest (the Docker runner's tool image)
```

See `spec/001_lumina_core/spec.md` for the full design spec.
