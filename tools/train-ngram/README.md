# train-ngram

Offline trainer for the n-gram language model used by `internal/aidetect`'s
perplexity scorer (see `spec/008_ai_detector/spec.md`). This tool is
standalone: it is never imported by the `lumina` binary, and it never runs as
part of `make build` or `make test`. Run it by hand when the corpus or
training parameters change, and commit the resulting model asset.

## Corpus

The model is trained on modern academic prose so that its notion of
"predictable" matches the register lumina actually scores: manuscript
paragraphs, not 19th-century novels — and not just abstracts, either.
`fetchcorpus` downloads, per category, both the paper title/abstract (from
the arXiv API) and the full LaTeX source of up to `fullTextPerCategory`
papers (via arXiv's e-print endpoint, stripped to plain prose by
`fetchcorpus/latex.go`), into `corpus/` (gitignored — the raw corpus is not
committed, only the trained model is). Eleven categories span vocabulary
breadth:

cs.CL, cs.LG, cs.AI, math.CO, math.ST, physics.gen-ph, astro-ph.GA, q-bio.GN,
q-bio.NC, stat.ME, econ.GN

```sh
go run ./fetchcorpus -out corpus
```

The fetch is restricted to papers **submitted on or before 2023-12-31**.
ChatGPT launched in November 2022; capping the corpus there keeps LLM-assisted
writing from leaking into what's supposed to be a human-prose baseline. It
also respects arXiv's API etiquette (a 3-second pause between requests), so
expect a full fetch (abstracts plus ~40 full papers per category) to take
25-35 minutes. A transient network error on one category is logged and
skipped rather than aborting the whole run — `fetchCategory` skips any
category whose output file already exists, so re-running the same command
resumes from the first category that never finished.

Full-text prose still won't cover vocabulary outside general academic
writing: a manuscript full of domain-specific proper nouns (product names,
identifiers, jargon from a narrow technical field) will read as high
perplexity regardless of authorship, because that vocabulary is genuinely
absent from the training corpus. That's a real, load-bearing limit of an
offline, general-purpose n-gram model — see the Calibration note below and
`internal/aidetect/scorer.go`'s comment on `perplexityHigh`.

## Training

```sh
go run . -corpus corpus -out ../../internal/aidetect/assets/model.bin.gz
```

Flags (all optional, defaults shown):

- `-order 3`: highest n-gram order to train (3 = trigrams).
- `-max-vocab 20000`: vocabulary cap; words outside the cap map to `<unk>`.
- `-min-context-count 2`: drop contexts observed fewer than this many times.
- `-max-continuations 50`: keep at most this many continuations per context.

## Model format

The trainer writes a gob-encoded `aidetect.Model` struct (defined in
`internal/aidetect/ngram.go`, the single source of truth for the format),
gzip compressed:

- `Vocab`: word index -> token, index 0 reserved for `<unk>`.
- `Unigram` / `TotalUnigram`: raw unigram counts.
- `Contexts`: for each `(order):(context word indices)` key, the pruned list
  of continuations and their counts.

`internal/aidetect/ngram.go` decodes the same struct via `embed.FS` at
program start; keep the two in sync if the format changes.

## Regenerating

The committed `internal/aidetect/assets/model.bin.gz` is not regenerated at
build or run time. To regenerate it after changing the corpus or trainer:

```sh
go run ./fetchcorpus -out corpus
go run . -corpus corpus -out ../../internal/aidetect/assets/model.bin.gz
```

## Calibration

The composite scorer's perplexity thresholds (`perplexityLow`/`perplexityHigh`
in `internal/aidetect/scorer.go`) are tuned against this specific model's
output range, not a universal constant. If you retrain against a
meaningfully different corpus, re-measure and re-check those thresholds:
generic LLM-stock-phrase filler should land near the low end (currently
~75-165), plain unadorned human sentences in the low thousands, and human
prose using vocabulary the corpus doesn't cover higher still — set
`perplexityHigh` conservatively above that last range, since perplexity
alone should never be the signal that flags jargon-heavy technical writing.
