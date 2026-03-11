# AI-Only Guardrails (Pilot)

Status: active pilot policy

Purpose: define what AI-only mutation lanes are allowed to change, what is hard denied, and what rollback evidence is mandatory.

## Allowed Scope Matrix

| Lane | Allowed repos/examples | Allowed mutation class | Required controls |
|---|---|---|---|
| AI-only prompt/workflow lane | `swamp-automation`, `ops-workflow` | Workflow-step edits, schedule/window edits, model-method wiring | Same repo/render boundary, hard-deny scan, rollback/revert hook required, governed decision state (`ALLOW/ESCALATE/BLOCK`) |

Default allowed repo list in scripts:

- `AI_ONLY_ALLOWED_REPOS=swamp-automation,ops-workflow`

## Hard Deny List

AI-only lanes must fail when workflow content includes high-risk patterns such as:

- `cluster-admin`
- `system:masters`
- `deleteEverything`
- `delete namespace`

Default deny regex in scripts:

- `AI_ONLY_HARD_DENY_REGEX=cluster-admin|system:masters|\bdeleteEverything\b|\bdelete\s+namespace\b`

## Mandatory Rollback Hooks

Every AI-only workflow lane must include at least one rollback/revert hook before execution.

Required by default:

- `AI_ONLY_REQUIRE_ROLLBACK_HOOK=1`

Hook detection rule:

- at least one YAML line matches `rollback` or `revert`

If missing, scripts fail before import/publish with remediation text.

## Enforced Paths

Guardrail helper:

- `examples/demo/lib/ai-only-guardrails.sh`

Guarded scripts:

- `examples/demo/prompt-as-dry-local.sh`
- `examples/demo/prompt-as-dry-connected.sh`

CI gate:

- `test/checks/check-ai-only-scope.sh`

The CI gate fails on:

1. out-of-scope repo attempts in AI-only lane
2. missing rollback/revert hook

## Operational Note

This policy is for the AI-only pilot lane only. Mixed human/CI/AI flows (for example Story 12) remain in governed collaborative mode and are not reclassified as AI-only.
