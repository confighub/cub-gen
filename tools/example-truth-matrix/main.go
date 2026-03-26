package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/confighub/cub-gen/internal/exampletruth"
)

func main() {
	var format string
	var root string
	flag.StringVar(&format, "format", "json", "output format: json or markdown")
	flag.StringVar(&root, "root", ".", "repository root")
	flag.Parse()

	matrix, err := exampletruth.Collect(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(matrix); err != nil {
			fmt.Fprintf(os.Stderr, "error: encode json: %v\n", err)
			os.Exit(1)
		}
	case "markdown", "md":
		fmt.Print(renderMarkdown(matrix))
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported format %q\n", format)
		os.Exit(1)
	}
}

func renderMarkdown(matrix exampletruth.Matrix) string {
	var b strings.Builder

	b.WriteString("# Example Truth Matrix\n\n")
	b.WriteString("Generated from repo structure, source-side tests, connected runners, and live-proof harness scripts. Do not edit by hand; regenerate with `go run ./tools/example-truth-matrix --format markdown`.\n\n")
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "- Featured examples: `%d`\n", matrix.Summary.FeaturedExamples)
	fmt.Fprintf(&b, "- Generator fixtures: `%d`\n", matrix.Summary.GeneratorFixtures)
	fmt.Fprintf(&b, "- Source-chain verified: `%d`\n", matrix.Summary.SourceChainVerified)
	fmt.Fprintf(&b, "- Connected mode present: `%d`\n", matrix.Summary.ConnectedModePresent)
	fmt.Fprintf(&b, "- Connected release gated: `%d`\n", matrix.Summary.ConnectedReleaseGated)
	fmt.Fprintf(&b, "- Real live proof: `none=%d`, `paired-harness=%d`, `standalone=%d`\n",
		matrix.Summary.RealLiveProof[exampletruth.RealLiveNone],
		matrix.Summary.RealLiveProof[exampletruth.RealLivePairedHarness],
		matrix.Summary.RealLiveProof[exampletruth.RealLiveStandalone],
	)
	fmt.Fprintf(&b, "- AI-first surface: `none=%d`, `partial=%d`, `explicit=%d`\n\n",
		matrix.Summary.AIFirstSurface[exampletruth.AIFirstNone],
		matrix.Summary.AIFirstSurface[exampletruth.AIFirstPartial],
		matrix.Summary.AIFirstSurface[exampletruth.AIFirstExplicit],
	)

	b.WriteString("## Matrix\n\n")
	b.WriteString("| Example | Generator fixture | Source chain verified | Connected mode | Connected release gate | Real live proof | AI-first surface | Tracking issues |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, row := range matrix.Rows {
		fmt.Fprintf(
			&b,
			"| `%s` | %s | %s | %s | %s | `%s` | `%s` | %s |\n",
			row.Example,
			yesNo(row.GeneratorFixture),
			yesNo(row.SourceChainVerified),
			yesNo(row.ConnectedModePresent),
			yesNo(row.ConnectedReleaseGated),
			row.RealLiveProof,
			row.AIFirstSurface,
			renderIssueLinks(row.TrackingIssues),
		)
	}

	b.WriteString("\n## Proof References\n\n")
	for _, row := range matrix.Rows {
		fmt.Fprintf(&b, "### `%s`\n\n", row.Example)
		writeRefs(&b, "Source chain", row.ProofRefs.SourceChain)
		writeRefs(&b, "Connected mode", row.ProofRefs.ConnectedMode)
		writeRefs(&b, "Connected release gate", row.ProofRefs.ConnectedReleaseGate)
		writeRefs(&b, "Real live", row.ProofRefs.RealLive)
		writeRefs(&b, "AI-first", row.ProofRefs.AIFirst)
		if len(row.Notes) > 0 {
			fmt.Fprintf(&b, "- Notes: %s\n", strings.Join(row.Notes, " "))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func writeRefs(b *strings.Builder, label string, refs []string) {
	if len(refs) == 0 {
		fmt.Fprintf(b, "- %s: --\n", label)
		return
	}
	fmt.Fprintf(b, "- %s: ", label)
	for i, ref := range refs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "`%s`", ref)
	}
	b.WriteString("\n")
}

func renderIssueLinks(issues []string) string {
	linked := make([]string, 0, len(issues))
	for _, issue := range issues {
		number := strings.TrimPrefix(issue, "#")
		linked = append(linked, fmt.Sprintf("[#%s](https://github.com/confighub/cub-gen/issues/%s)", number, number))
	}
	return strings.Join(linked, ", ")
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
