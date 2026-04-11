# AI Example Hygiene Checklist

Use this checklist when deciding whether a workflow or AI-oriented example is
ready to represent the product surface.

This is intentionally short. It is the repo-level "good hygiene" bar for
examples where the first operator may be an AI assistant using the CLI.

## Core checklist

- State the stack and operator in the first few lines.
- Add a `What this proves today` table.
- Distinguish source-side proof, connected governance proof, and live/runtime proof.
- Say what is still not proven yet.
- Give one `local` first run and one `connected` first run.
- Start with the read-only or lowest-risk path before mutation.
- Make it obvious what the example reads, writes, and does not write.
- Show one concrete `ALLOW` path and one concrete `ESCALATE` or `BLOCK` path.
- Tell the operator what to inspect after each major step.
- Point back to the generated [Example Truth Matrix](../testing/example-truth-matrix.md) when proof level matters.

## Extra requirements for AI-authored or non-deterministic flows

- Treat prompt/context as DRY input when that is the real authoring surface.
- Explain that verification and attestation are the safety boundary.
- Call out mutation-ledger or evidence artifacts as the audit trail.
- Say clearly whether the example proves structural governance, real runtime execution, or both.
- If the example is only a companion or illustrative path today, say so directly.

## Preferred example shape

1. Short scenario intro
2. `What this proves today`
3. Fastest first run
4. Who this is for
5. Why `cub-gen` plus ConfigHub helps
6. Concrete `ALLOW` / `BLOCK` examples
7. Local and connected entrypoints
8. Known gaps and companion examples

## Related guidance

- [Prompt as DRY](prompt-as-dry.md)
- [AI-only guardrails](ai-only-guardrails.md)
- [Example Truth Matrix](../testing/example-truth-matrix.md)
