# lumina

Lumina is a standalone CLI for the academic-writing pipeline: Mermaid
diagrams, citation checking, bibliography pruning/formatting, prose linting,
word-count enforcement, and PDF/DOCX/TeX/ZIP builds — all driven from a
project directory managing multiple manuscript targets.

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
mkdir my-project && cd my-project
lumina init paper1           # scaffold project layout and src/paper1/ target
lumina build paper1 --pdf    # -> build/paper1.pdf
```

## Directory layout

Lumina is invoked from within a project root directory; all targets reside in `src/`.

| Path                               | Purpose |
|------------------------------------|---------|
| `lumina.yaml`                      | Project-level tool/environment configuration. |
| `csl/`                             | Shared CSL bibliography style sheets across all targets. |
| `templates/<template-name>/`       | Named template folders containing `template.tex`, `*.sty`, `*.cls`, `*.bst`, or `reference.docx`. |
| `src/<target>/manuscript.md`       | Manuscript source prose for the target. |
| `src/<target>/metadata.yaml`       | Target manuscript metadata, template selection, and output settings. |
| `src/<target>/references.bib`      | BibTeX database for the target. |
| `src/<target>/figures/`            | Static figures for the target. |
| `src/<target>/literature/`         | User-managed PDF library with same-stem `.bib` sidecars (optional). |
| `.vale.ini`                        | Vale prose-linter config in project root. |
| `.lumina/`                         | Common temporary staging directory. Cleared before every build. |
| `build/`                           | Final build output artifacts (`<output>.pdf`, `<output>.docx`, etc.). |

`lumina init <target>` scaffolds project root files if absent, plus `src/<target>/`
files and directories. It never overwrites existing files.

## Commands

All commands are run from the project root.

### Top-level

| Command | Description |
|---------|-------------|
| `lumina init <target>` | Scaffold project root files (if absent) and a new target under `src/<target>/`. Never overwrites. |
| `lumina clean` | Remove `.lumina/` and `build/`. Leaves targets and project config untouched. |

### `lumina build` — compilation

| Command | Description |
|---------|-------------|
| `lumina build <target>` | Build all formats configured in `lumina.yaml`'s `formats` list (default: `pdf docx tex zip`). |
| `lumina build <target> --pdf` | Build PDF (`build/<output>.pdf`). |
| `lumina build <target> --docx` | Build Word document (`build/<output>.docx`). |
| `lumina build <target> --tex` | Build standalone LaTeX source (`build/<output>.tex`). |
| `lumina build <target> --zip` | Build ZIP submission archive (`build/<output>.zip`). |
| `lumina build <target> --pub` | Pre-submission gate: citation check → Vale lint → word-limit check → TODO scan, all fail-fast. On success, builds PDF + ZIP dated release files in `build/`. |
| `lumina build <target> --preprocess` | Run only preprocessing and diagram rendering. |

Flags:
* `--pdf`, `--docx`, `--tex`, `--zip`: Select specific formats to build.
* `--pub`: Run publication validation gates and release build.
* `--force` / `-f`: Force re-rendering of Mermaid diagrams and bypass caches.
* `--pdf-engine ENGINE`: Override pandoc's `--pdf-engine` (e.g. `xelatex`, `lualatex`).

### `lumina text` — prose quality

| Command | Description |
|---------|-------------|
| `lumina text words <target>` | Word count via `pandoc --to=plain`. Reports `count / limit` if `wordlimit` is set in `metadata.yaml`. |
| `lumina text fmt <target>` | Format `src/<target>/manuscript.md` with prettier. |
| `lumina text lint <target>` | Runs Vale against `src/<target>/manuscript.md` using `.vale.ini` from project root. |

### `lumina lit` — literature & bibliography

| Command | Description |
|---------|-------------|
| `lumina lit check <target>` | Verifies every `@key` cited in `src/<target>/manuscript.md` has a matching entry in `src/<target>/references.bib`. |
| `lumina lit prune <target> [--no-dry-run] [--yes/-y]` | Removes bibliography entries not cited in the target manuscript. **Dry-run by default**. |
| `lumina lit fmt <target>` | Formats `src/<target>/references.bib` in place. |

## Configuration

### `src/<target>/metadata.yaml`

Manuscript metadata:

- **Pandoc-standard keys**, forwarded to pandoc verbatim: `title`, `author`, `date`, `bibliography`, `csl`, `numbersections`, `geometry`, etc.
- **Lumina-specific keys**, stripped before forwarding:
  - `output`: Output base name for artifacts in `build/` (defaults to `<target>` if omitted).
  - `template`: Name of template folder under `templates/<template>/`.
  - `wordlimit`: Integer word cap (`0` = unlimited).
- **Reshaped-and-forwarded**: `acronyms`. Flat `KEY: "definition"` map reshaped into `pandoc-acro`'s schema.

```yaml
title:          "Untitled Manuscript"
author:         "Author Name"
date:           "2026-09-22"
bibliography:   "references.bib"
csl:            "ieee.csl"
numbersections: true

template:  "default"
output:    "my-paper"
wordlimit: 8000

acronyms:
  API: "Application Programming Interface"
  CLI: "Command Line Interface"
```

### `lumina.yaml` (Project Root)

Tool/environment configuration — not manuscript metadata. All keys optional.

```yaml
pdf-engine:  xelatex                # pandoc --pdf-engine value
formats:                            # formats built by default
  - pdf
  - docx
  - tex
  - zip
runner:      host                   # host | docker
tools-image: lumina-tools:latest    # used when runner: docker
```

### Runner: host vs. Docker

Lumina never calls external tools directly — every invocation goes through
a `Runner`. With `runner: host` (default), tools run via the host's `PATH`.
With `runner: docker`, each tool call becomes its own
`docker run --rm -v <project-root>:/workspace -w /workspace <tools-image> <tool> <args...>`
against the image built by `make image` in this repo.

## Mermaid diagrams

Any ` ```mermaid ` fenced code block in `manuscript.md` is rendered to a
PNG during preprocessing. Blocks are cached by `SHA-256(block source)[:16]`
under `.lumina/figures/mermaid-<hash>.png`. Pass `--force` / `-f` to any
`build` invocation to bypass the cache and re-render everything.

## Developing lumina itself

```sh
make build     # -> _build/lumina
make test      # go test ./...
make vet       # go vet ./...
make install   # -> $HOME/.local/bin/lumina
make image     # build lumina-tools:latest (the Docker runner's tool image)
```

See `spec/007_multi_target_projects/spec.md` for the multi-target project design spec.
