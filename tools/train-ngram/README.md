# train-ngram

Offline trainer for the n-gram language model used by `internal/aidetect`'s
perplexity scorer (see `spec/008_ai_detector/spec.md`). This tool is
standalone: it is never imported by the `lumina` binary, and it never runs as
part of `make build` or `make test`. Run it by hand when the corpus or
training parameters change, and commit the resulting model asset.

## Corpus

The model is trained on modern academic prose so that its notion of
"predictable" matches the register lumina actually scores: manuscript
paragraphs, not 19th-century novels. `fetchcorpus` downloads paper titles and
abstracts from the arXiv API into `corpus/` (gitignored — the raw corpus is
not committed, only the trained model is), spanning eleven categories for
vocabulary breadth:

cs.CL, cs.LG, cs.AI, math.CO, math.ST, physics.gen-ph, astro-ph.GA, q-bio.GN,
q-bio.NC, stat.ME, econ.GN

```sh
go run ./fetchcorpus -out corpus
```

The fetch is restricted to papers **submitted on or before 2023-12-31**.
ChatGPT launched in November 2022; capping the corpus there keeps LLM-assisted
writing from leaking into what's supposed to be a human-prose baseline. It
also respects arXiv's API etiquette (a 3-second pause between requests), so
expect the fetch to take a couple of minutes.

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
meaningfully different corpus, re-check those thresholds: genuinely human,
unseen academic prose should land near the high end (little to no
suspicion), and dense LLM stock phrasing near the low end.
