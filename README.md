# Lumina

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![Build Status](https://img.shields.io/badge/tests-passing-brightgreen.svg)]()
[![License](https://img.shields.io/badge/license-MIT-blue.svg)]()

Lumina is a standalone, deterministic CLI for academic writing pipelines. It orchestrates Markdown-based manuscript authoring across multiple targets, providing automated diagram compilation, citation verification, bibliography pruning, prose linting, statistical AI text detection, Git revision evolution tracking, and multi-format publication builds (PDF, DOCX, LaTeX, submission ZIP).

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
  - [Host Prerequisites](#host-prerequisites)
  - [Building from Source](#building-from-source)
  - [Docker Runner (Zero-Install Toolchain)](#docker-runner-zero-install-toolchain)
- [Quick Start](#quick-start)
- [Project Architecture & Directory Layout](#project-architecture--directory-layout)
- [Command Reference](#command-reference)
  - [Project Commands](#project-commands)
    - [`lumina init`](#lumina-init)
    - [`lumina clean`](#lumina-clean)
  - [Manuscript Commands](#manuscript-commands)
    - [`lumina build`](#lumina-build)
      - [Pre-submission Publication Gate (`--pub`)](#pre-submission-publication-gate---pub)
    - [`lumina log`](#lumina-log)
    - [`lumina text`](#lumina-text)
      - [Prose Quality & AI Detection Engine](#prose-quality--ai-detection-engine)
    - [`lumina lit`](#lumina-lit)
- [Configuration Reference](#configuration-reference)
  - [`lumina.yaml` (Project Level)](#luminayaml-project-level)
  - [Environment Variables (`.env`)](#environment-variables-env)
  - [`metadata.yaml` (Target Level)](#metadatayaml-target-level)
  - [Vale & Vocabulary Management](#vale--vocabulary-management)
- [Advanced Features](#advanced-features)
  - [Mermaid Diagram Acceleration & Caching](#mermaid-diagram-acceleration--caching)
  - [LaTeX Templates & Multi-File Style Packages](#latex-templates--multi-file-style-packages)
  - [Acronym Reshaping (`pandoc-acro`)](#acronym-reshaping-pandoc-acro)
- [Development](#development)

---

## Overview

Academic manuscript authoring typically suffers from fragmented toolchains: Makefile spaghetti, inconsistent LaTeX dependencies, broken citation references, manual Word conversions for co-authors, and opaque review cycles.

Lumina provides a single, unified interface for research projects:

- **Multi-Target Isolation**: Maintain papers, conference submissions, rebuttal letters, and thesis chapters in one project while sharing templates, CSL citation stylesheets, and vocabulary dictionaries.
- **Multi-Format Generation**: Compile publication-grade PDFs via XeLaTeX or LuaLaTeX, structured Word documents (`.docx`), standalone standalone LaTeX source trees (`.tex`), and camera-ready submission archives (`.zip`).
- **Enforced Integrity Gates (`--pub`)**: Hard pre-submission validation preventing broken `@citations`, unexpanded TODO notes, style guide violations, and word cap overflows.
- **Offline AI Prose Detection**: 100% local, deterministic heuristics (n-gram perplexity, sentence burstiness, lexical density, stock phrasing) auditing manuscripts without sending research to third-party cloud APIs.
- **Git Prose Evolution Tracking**: Extract paragraph-level revision diffs, calculated writing duration, and citation alterations directly from repository history into standalone HTML, Markdown, PDF, or terminal ANSI reports.
- **Dual Execution Engine**: Run natively against local host binaries or completely containerized through an isolated Docker toolchain image.

---

## Installation

### Host Prerequisites

When running in host mode (`runner: host`), Lumina orchestrates external tools found on your `$PATH`:

| Tool | Purpose | Recommended Version |
| :--- | :--- | :--- |
| **Pandoc** | Markdown AST transformations and format conversion | `>= 3.1` |
| **pandoc-crossref** | Numbered cross-referencing (figures, equations, tables) | Compatible with Pandoc |
| **pandoc-acro** | Acronym expansion and glossary indexing | Latest |
| **TeX Live / MikTeX** | PDF rendering engines (`xelatex`, `lualatex`, `pdflatex`) | TeX Live `>= 2023` |
| **mmdc** (`@mermaid-js/mermaid-cli`) | Automated Mermaid code block diagram rendering | `>= 10.0` |
| **Vale** | Prose style, grammar, and editorial rules linter | `>= 3.0` |
| **Prettier** | Opinionated Markdown formatting | `>= 3.0` |
| **zip** | Submission archive packaging | System standard |

### Building from Source

Ensure Go 1.22+ is installed on your workstation:

```sh
# Clone repository
git clone https://github.com/DigitalEntity/lumina.git
cd lumina

# Build binary into ./_build/lumina
make build

# Install binary to $HOME/.local/bin/lumina
make install
```

Make sure `$HOME/.local/bin` is present in your `$PATH`.

### Docker Runner (Zero-Install Toolchain)

To avoid installing TeX Live, Pandoc filters, Node.js, and Vale on your host system, build the containerized toolchain image:

```sh
make image    # Builds lumina-tools:latest
```

Set the execution runner to `docker` in your `lumina.yaml`:

```yaml
runner: docker
tools-image: lumina-tools:latest
```

Every external invocation runs inside an isolated container mounting the project root to `/workspace`.

---

## Quick Start

### 1. Initialize a Project and Target

```sh
mkdir my-research && cd my-research
lumina init paper1
```

This creates the project-level scaffolding (`lumina.yaml`, `.vale.ini`, `csl/`, `templates/default/`, `vocab/Default/`) and your initial target directory `src/paper1/`.

### 2. Add Content

- Edit `src/paper1/manuscript.md` with your prose and citations:
  ```markdown
  # Introduction

  Recent advances demonstrate significant throughput gains [@sharlaimov2026].

  ```mermaid
  graph TD
      A[Raw Data] --> B(Preprocessing)
      B --> C{Validation}
      C -->|Pass| D[Artifact]
  ```
  ```
- Add bibliographic entries to `src/paper1/references.bib`.
- Configure target metadata in `src/paper1/metadata.yaml`.

### 3. Verify Prose and Citations

```sh
lumina lit check paper1      # Verify all @cite keys exist in references.bib
lumina text words paper1     # Check prose word count against configured limits
lumina text detect paper1    # Audit prose for machine-generation statistical markers
```

### 4. Compile Target Artifacts

```sh
lumina build paper1 --pdf    # Output: build/paper1.pdf
lumina build paper1 --docx   # Output: build/paper1.docx
lumina build paper1 --zip    # Output: build/paper1.zip (submission bundle)
```

### 5. Audit Evolution

```sh
lumina log paper1 --terminal # Inspect paragraph-level changes across Git commits
lumina log paper1            # Generate visual HTML report in build/paper1-changelog.html
```

---

## Project Architecture & Directory Layout

Lumina projects organize files into project-wide shared configurations and isolated manuscript targets:

```
my-research/
├── lumina.yaml                     # Project-level tool & runner configuration
├── .env                            # Environment variables (optional)
├── .vale.ini                       # Vale prose linter configuration
├── csl/                            # Citation Style Language sheets (e.g. ieee.csl)
│   └── ieee.csl
├── templates/                      # LaTeX and Word templates
│   └── default/
│       ├── template.tex            # Pandoc LaTeX template
│       ├── reference.docx          # Word style reference document
│       └── *.sty, *.cls, *.bst     # LaTeX style and class packages
├── vocab/                          # Project-wide Vale custom dictionaries
│   └── Default/
│       ├── accept.txt              # Accepted terminology (allowed technical jargon)
│       └── reject.txt              # Prohibited vocabulary
├── src/                            # Isolated manuscript targets
│   ├── paper1/
│   │   ├── manuscript.md           # Target prose source
│   │   ├── metadata.yaml           # Target pandoc and build metadata
│   │   ├── references.bib          # Target BibTeX database
│   │   ├── accept.txt              # Target-specific allowed vocabulary (optional)
│   │   ├── figures/                # Static images and charts
│   │   └── literature/             # Research PDFs and .bib sidecars (optional)
│   └── thesis-ch1/
│       ├── manuscript.md
│       ├── metadata.yaml
│       └── references.bib
├── .lumina/                        # Staging & build cache (auto-managed)
│   ├── build/                      # Clean intermediate target workspace
│   └── figures/                    # Content-hashed Mermaid PNG cache
└── build/                          # Final compiled output artifacts
    ├── paper1.pdf
    ├── paper1.docx
    ├── paper1.tex
    ├── paper1.zip
    └── paper1-changelog.html
```

---

## Command Reference

### Project Commands

#### `lumina init`

Scaffold project root files and an initial manuscript target under `src/<target>/`. Existing files are never overwritten.

```sh
lumina init <target>
```

**Created Assets:**
- Root: `lumina.yaml`, `.vale.ini`, `.gitignore`, `csl/.gitkeep`, `templates/default/.gitkeep`, `vocab/Default/{accept.txt,reject.txt}`.
- Target: `src/<target>/manuscript.md`, `metadata.yaml`, `references.bib`, `figures/.gitkeep`, `literature/.gitkeep`.

#### `lumina clean`

Purge temporary staging caches (`.lumina/`) and compiled artifacts (`build/`). Leaves target source files and project configurations untouched.

```sh
lumina clean
```

---

### Manuscript Commands

#### `lumina build`

Compile a manuscript target into one or more output formats.

```sh
lumina build <target> [flags]
```

**Flags:**

| Flag | Description |
| :--- | :--- |
| `--pdf` | Compile target to PDF (`build/<output>.pdf`). |
| `--docx` | Compile target to Microsoft Word (`build/<output>.docx`). |
| `--tex` | Export standalone, self-contained LaTeX source (`build/<output>.tex`). |
| `--zip` | Package submission archive with `.tex`, `.bib`, styles, and figures (`build/<output>.zip`). |
| `--pub` | Execute pre-submission publication gates and produce dated release artifacts. |
| `--preprocess` | Execute preprocessing, diagram compilation, and staging without compiling targets. |
| `-f, --force` | Bypass cached intermediate assets and force re-rendering of all Mermaid diagrams. |
| `--pdf-engine <engine>` | Override the default PDF engine (e.g. `xelatex`, `lualatex`, `pdflatex`). |

If no format flag is provided, Lumina builds all formats specified in `lumina.yaml`'s `formats` array.

##### Pre-submission Publication Gate (`--pub`)

The `--pub` flag executes four strict, fail-fast verification gates before compiling publication assets:

1. **Citation Check**: Verifies that every citation `@key` in `manuscript.md` resolves against `references.bib`.
2. **Prose Linting**: Runs Vale against project and target vocabulary dictionaries.
3. **Word Limit Gate**: Renders plain text and checks word count against `wordlimit` configured in `metadata.yaml`.
4. **TODO Sentinel Scan**: Scans prose for unfinished markers (`TODO` or `{.todo}`).

Upon passing all gates, Lumina generates canonical artifacts (`<output>.pdf`, `<output>.zip`) as well as timestamped archive copies (`<output>_YYYY-MM-DD.pdf`, `<output>_YYYY-MM-DD.zip`).

---

#### `lumina log`

Inspect the Git version control history of `src/<target>/manuscript.md` and `references.bib` to produce an evolution changelog. Demonstrates iterative human progression and drafting milestones.

```sh
lumina log <target> [flags]
```

**Flags:**

| Flag | Description |
| :--- | :--- |
| `-o, --output <path>` | Custom destination path (default: `build/<output>-changelog.html`). |
| `-m, --markdown, --md` | Export changelog in Markdown format (`build/<output>-changelog.md`). |
| `-t, --terminal` | Render colored ANSI diff output directly to standard output. |
| `--stat` | Output tabular revision summary statistics without paragraph diff details. |
| `--pdf` | Compile changelog to PDF via Pandoc (`build/<output>-changelog.pdf`). |
| `--since <date>` | Filter revisions after a specific date or Git ref (e.g. `2026-01-01`, `RFC3339`). |
| `-n, --max-count <N>` | Restrict analysis to the most recent $N$ revisions (default: `0`, all). |

**Rendered Elements:**
- **Paragraph Revisions**: Word-level diffing rendering deleted words with red strikethrough (`<del>`) and added words with green highlighting (`<ins>`).
- **Bibliography Tracking**: Identifies added or altered bibliography entries and renders them as formatted citations.
- **Milestone Navigation**: Sticky header with direct commit jumping, net word counts, and elapsed drafting duration.

---

#### `lumina text`

Prose auditing, formatting, and quality enforcement.

##### `lumina text words <target>`
Extracts rendered plain-text prose via Pandoc and counts words. Compares count against `wordlimit` configured in `metadata.yaml`.

##### `lumina text fmt <target>`
Formats `src/<target>/manuscript.md` in-place using Prettier for clean, consistent Markdown structure.

##### `lumina text lint <target>`
Lints prose style against `.vale.ini` rules, automatically syncing custom vocabularies from `vocab/` and `src/<target>/`.

##### `lumina text detect <target>` (Alias: `ai`)
Audits manuscript prose for statistical signatures of automated language model generation. Runs completely offline with no network connections or telemetry.

```sh
lumina text detect <target> [flags]
```

**Flags:**

| Flag | Description |
| :--- | :--- |
| `-t, --threshold <N>` | Minimum composite suspicion score (0–100) to flag a paragraph (default: `60`). |
| `-d, --detail` | Display individual metric breakdowns for flagged paragraphs. |
| `-j, --json` | Emit complete analysis report as machine-readable JSON. |

##### Prose Quality & AI Detection Engine

The detector isolates prose paragraphs using Goldmark AST analysis (stripping math formulas, code blocks, raw HTML, tables, and citation keys) and evaluates:

- **Perplexity / Predictability**: Low token cross-entropy against an embedded n-gram model trained on pre-2024 academic writing.
- **Sentence Burstiness**: Variance of sentence lengths. Uniform sentence lengths signal automated text; high variance indicates human drafting.
- **Lexical Diversity**: Type-Token Ratio (TTR) measuring vocabulary repetition.
- **Stock-Phrase Density**: Pattern matching against characteristic LLM stock phrases (*"delve"*, *"testament"*, *"pivotal"*, *"furthermore"*, *"crucial role"*).
- **Hedging & Punctuation**: Overuse of em-dashes and soft hedging markers (*"it is worth noting"*, *"generally speaking"*).

---

#### `lumina lit`

Literature and bibliography management.

##### `lumina lit check <target>`
Verifies that every citation `@key` in `manuscript.md` has an existing entry in `references.bib`. Ignores Pandoc cross-reference identifiers (such as `@fig:`, `@tbl:`, `@eq:`, `@sec:`).

##### `lumina lit prune <target>`
Identifies and strips unused bibliographic entries from `src/<target>/references.bib`.

- **Dry-run by default**: Reports unreferenced entries without modifying disk files.
- Pass `--no-dry-run` to rewrite `references.bib`.
- Pass `--yes` / `-y` to bypass the interactive confirmation prompt.

```sh
lumina lit prune paper1 --no-dry-run --yes
```

##### `lumina lit fmt <target>`
Formats and normalizes `src/<target>/references.bib` in-place with alphabetical key sorting and consistent field indentation.

---

## Configuration Reference

### `lumina.yaml` (Project Level)

Placed in the project root. Defines global build behavior and execution parameters:

```yaml
# PDF compilation engine executed by Pandoc
pdf-engine: xelatex                  # Options: xelatex | lualatex | pdflatex

# Default formats compiled when running `lumina build <target>` without flags
formats:
  - pdf
  - docx
  - tex
  - zip

# Toolchain execution mode: host (local PATH) or docker (isolated container)
runner: host                         # Options: host | docker
tools-image: lumina-tools:latest     # Used when runner is 'docker'

# Configuration for prose auditing tools
text:
  detect:
    threshold: 60                    # Default suspicion threshold (0-100)
    ignore_phrases:                  # Phrases exempted from stock-phrase penalties
      - "in conclusion"
      - "it should be noted that"
```

### Environment Variables (`.env`)

Lumina automatically reads a `.env` file located in the project root upon invocation. Existing environment variables are not overwritten:

```sh
# .env
PANDOC_PDF_ENGINE=lualatex
VALE_STYLES_PATH=.lumina/styles
```

### `metadata.yaml` (Target Level)

Placed in `src/<target>/metadata.yaml`. Separates Lumina orchestration keys from Pandoc frontmatter:

```yaml
# ==============================================================================
# Lumina Specific Keys (Stripped before forwarding to Pandoc)
# ==============================================================================
output: "icse2026-paper"             # Stem name for output files in build/
template: "default"                  # Template folder under templates/<template>/
wordlimit: 8500                      # Word limit cap (0 = unlimited)

# ==============================================================================
# Acronym Declarations (Reshaped automatically for pandoc-acro)
# ==============================================================================
acronyms:
  AST: "Abstract Syntax Tree"
  CLI: "Command Line Interface"
  LCS: "Longest Common Subsequence"

# ==============================================================================
# Standard Pandoc Metadata (Forwarded verbatim to Pandoc)
# ==============================================================================
title: "Deterministic Compilation for Academic Manuscripts"
author:
  - name: "Konstantin Sharlaimov"
    affiliation: "Engineering Division"
date: "2026-09-24"
bibliography: "references.bib"
csl: "ieee.csl"
numbersections: true
link-citations: true
geometry: "margin=1in"
```

### Vale & Vocabulary Management

Lumina orchestrates Vale prose linting using `.vale.ini` in the project root:

```ini
StylesPath = .lumina/styles
MinAlertLevel = suggestion

[*.md]
BasedOnStyles = Vale, write-good, proselint
```

#### Custom Dictionaries

To prevent false alarms on domain terminology or project jargon:

- **Project-Wide Vocabulary**: Place terms in `vocab/Default/accept.txt` (accepted jargon) and `vocab/Default/reject.txt` (prohibited words).
- **Target-Specific Vocabulary**: Place terms in `src/<target>/accept.txt`.
- During build and linting, Lumina automatically stages and merges these files into `.lumina/styles/config/vocabularies/` before executing Vale.

---

## Advanced Features

### Mermaid Diagram Acceleration & Caching

Lumina compiles fenced ` ```mermaid ` code blocks in `manuscript.md` directly into figures:

1. **Extraction**: Preprocessing identifies Mermaid diagram blocks.
2. **Persistent Hashing**: Computes `SHA-256(diagram_source)[:16]`.
3. **Cache Storage**: Stored under `.lumina/figures/mermaid-<hash>.png`. Unchanged diagrams are reused across builds.
4. **AST Replacement**: Pandoc receives standard Markdown image references (`![](figures/mermaid-<hash>.png)`).
5. **Cache Bypass**: Supply `--force` / `-f` to any build command to invalidate cached diagrams and force re-rendering.

### LaTeX Templates & Multi-File Style Packages

When targeting journals or conferences with proprietary formatting classes (e.g. IEEE, ACM, Springer), place style assets in `templates/<template-name>/`:

```
templates/acmart/
├── template.tex
├── acmart.cls
├── ACM-Reference-Format.bst
└── sample-base.eps
```

Lumina detects and automatically stages all supporting files matching `.sty`, `.cls`, `.bst`, `.bbx`, `.cbx`, `.dbx`, `.def`, `.cfg`, `.png`, `.jpg`, `.jpeg`, `.eps`, and `.pdf` into the staging directory before compilation.

### Acronym Reshaping (`pandoc-acro`)

Lumina allows authors to specify acronyms cleanly in `metadata.yaml`:

```yaml
acronyms:
  API: "Application Programming Interface"
  TTR: "Type-Token Ratio"
```

During preprocessing, Lumina reshapes this map into the schema required by the `pandoc-acro` filter:

```yaml
acronyms:
  API:
    short: API
    long: "Application Programming Interface"
  TTR:
    short: TTR
    long: "Type-Token Ratio"
```

Reference them in prose via `\+API` or `[+API]` for automated first-use expansion and subsequent abbreviated rendering.

---

## Development

### Makefile Targets

```sh
make build      # Compile lumina binary into _build/lumina
make test       # Run all package test suites (go test ./...)
make vet        # Run Go static analysis (go vet ./...)
make install    # Install compiled binary to $HOME/.local/bin/lumina
make image      # Build the Docker container tools image (lumina-tools:latest)
```

### Running Specific Tests

```sh
# Test citation integrity verification
go test -v ./internal/citations/...

# Test offline AI detection scoring engine
go test -v ./internal/aidetect/... -run TestScorer

# Test Git changelog diff extraction
go test -v ./internal/changelog/...
```

---

## License

MIT License. See `LICENSE` for details.
