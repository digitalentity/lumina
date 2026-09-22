// Rough LaTeX-to-plain-text stripping for full-paper corpus extraction. Not
// a real LaTeX parser: good enough to leave flowing prose behind for n-gram
// training, not to typeset or losslessly round-trip. Leftover artifacts
// (stray braces, an occasional macro name) are acceptable noise at corpus
// scale; the goal is dropping non-prose content (math, tables, citations,
// preamble) wholesale, not perfect fidelity.
package main

import (
	"regexp"
	"strings"
)

// dropEnvironments are LaTeX environments whose content is not prose and
// should be discarded entirely, including the \begin/\end markers.
var dropEnvironments = []string{
	"table", "table*", "figure", "figure*", "tabular", "tabular*",
	"equation", "equation*", "align", "align*", "eqnarray", "eqnarray*",
	"algorithm", "algorithmic", "verbatim", "lstlisting", "tikzpicture",
	"thebibliography", "array",
}

// dropWithArgCommands are commands whose argument is not prose (labels,
// references, citations, metadata, media) and should be removed along with
// their arguments.
var dropWithArgCommands = []string{
	"citep", "citet", "citealp", "citealt", "citeauthor", "citeyear", "cite",
	"ref", "eqref", "label", "footnote", "thanks", "includegraphics", "caption",
	"url", "href", "bibliography", "bibliographystyle", "title", "author",
	"affil", "date", "section", "subsection", "subsubsection", "paragraph",
	"pageonefooter", "dochead", "jvol", "jnum", "jyear", "runningtitle",
	"runningauthor", "usepackage", "documentclass", "newcommand",
	"renewcommand", "DeclareMathOperator", "input", "bibinput",
}

// unwrapCommands are formatting commands whose single-brace argument is
// itself prose and should be kept, dropping only the command wrapper.
var unwrapCommands = []string{"emph", "textbf", "textit", "underline", "texttt", "textsc"}

var (
	commentRe    = regexp.MustCompile(`(^|[^\\])%.*`)
	documentRe   = regexp.MustCompile(`(?s)\\begin\{document\}(.*)\\end\{document\}`)
	iffalseRe    = regexp.MustCompile(`(?s)\\iffalse.*?\\fi`)
	displayMathA = regexp.MustCompile(`(?s)\\\[.*?\\\]`)
	displayMathB = regexp.MustCompile(`(?s)\$\$.*?\$\$`)
	inlineMathA  = regexp.MustCompile(`(?s)\\\(.*?\\\)`)
	inlineMathB  = regexp.MustCompile(`\$[^$]*\$`)
	itemRe       = regexp.MustCompile(`\\item\s*(\[[^\]]*\])?`)
	bracesRe     = regexp.MustCompile(`[{}]`)
	multiSpaceRe = regexp.MustCompile(`[ \t]+`)
	multiBlankRe = regexp.MustCompile(`\n{3,}`)
)

// StripLatex reduces a LaTeX source file to its flowing prose: the document
// body with preamble, math, tables/figures, citations, and other non-prose
// markup removed.
func StripLatex(src string) string {
	src = stripComments(src)
	src = iffalseRe.ReplaceAllString(src, " ")

	if m := documentRe.FindStringSubmatch(src); m != nil {
		src = m[1]
	}

	for _, env := range dropEnvironments {
		src = dropEnvironment(src, env)
	}

	src = displayMathA.ReplaceAllString(src, " ")
	src = displayMathB.ReplaceAllString(src, " ")
	src = inlineMathA.ReplaceAllString(src, " ")
	src = inlineMathB.ReplaceAllString(src, " ")

	for _, cmd := range dropWithArgCommands {
		src = dropCommandWithArg(src, cmd)
	}
	for _, cmd := range unwrapCommands {
		src = unwrapCommand(src, cmd)
	}

	src = itemRe.ReplaceAllString(src, "")
	src = dropRemainingEnvironmentMarkers(src)
	src = dropUnknownCommands(src)
	src = bracesRe.ReplaceAllString(src, "")

	src = multiSpaceRe.ReplaceAllString(src, " ")
	src = multiBlankRe.ReplaceAllString(src, "\n\n")
	return strings.TrimSpace(src)
}

func stripComments(src string) string {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		lines[i] = commentRe.ReplaceAllString(line, "$1")
	}
	return strings.Join(lines, "\n")
}

// dropEnvironment removes every \begin{env}...\end{env} block (including
// the starred form via env already carrying the "*") for a fixed, known
// environment name.
func dropEnvironment(src, env string) string {
	re := regexp.MustCompile(`(?s)\\begin\{` + regexp.QuoteMeta(env) + `\}.*?\\end\{` + regexp.QuoteMeta(env) + `\}`)
	return re.ReplaceAllString(src, " ")
}

// dropRemainingEnvironmentMarkers strips \begin{...}/\end{...} wrappers for
// environments not in dropEnvironments (e.g. itemize, enumerate, abstract),
// keeping their inner content as prose.
func dropRemainingEnvironmentMarkers(src string) string {
	re := regexp.MustCompile(`\\(begin|end)\{[^}]*\}`)
	return re.ReplaceAllString(src, " ")
}

// dropCommandWithArg removes \cmd, \cmd[opts], \cmd{arg}, and
// \cmd[opts]{arg} forms (one optional-arg group, one required-arg group;
// LaTeX commands rarely nest more braces at this level for the commands in
// dropWithArgCommands). The \b after the command name stops "\cite" from
// swallowing the "\citep{...}"/"\citealp{...}" prefix and leaving the tail
// ("p{...}", "alp{...}") behind as garbage tokens.
func dropCommandWithArg(src, cmd string) string {
	re := regexp.MustCompile(`\\` + regexp.QuoteMeta(cmd) + `\b\*?(\[[^\]]*\])?(\{[^{}]*\})?`)
	return re.ReplaceAllString(src, " ")
}

// unwrapCommand replaces \cmd{text} with text, keeping the argument.
func unwrapCommand(src, cmd string) string {
	re := regexp.MustCompile(`\\` + regexp.QuoteMeta(cmd) + `\b\*?\{([^{}]*)\}`)
	return re.ReplaceAllString(src, "$1")
}

// dropUnknownCommands is the fallback pass: any remaining \commandname
// (with an optional [..] group) is removed, but a trailing {...} argument
// is kept as plain text, since most surviving macros at this point are
// custom text-formatting wrappers (\textcolor{red}{text}, journal-template
// macros, etc.) rather than structural markup.
func dropUnknownCommands(src string) string {
	re := regexp.MustCompile(`\\[a-zA-Z]+\*?(\[[^\]]*\])?`)
	return re.ReplaceAllString(src, " ")
}
