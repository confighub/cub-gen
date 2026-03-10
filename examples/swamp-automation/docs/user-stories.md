# Swamp Automation — User Stories

## 1. Agent proposes a workflow step

An agent adds `healthcheck` before `validate` in `workflow-deploy.yaml`.
`cub-gen` captures the structural change (new step + model/method reference),
produces a governed bundle, and checks policy inputs such as approved
model-method pairs.

## 2. Required safety step cannot be removed

A change removes the `validate` step. The policy declares `validate` as
required. The change should be surfaced as BLOCK/ESCALATE instead of silently
passing review.

## 3. Vault safety remains local-first

The team rotates vault keys in `.swamp.yaml`. Local verify/attest runs fast and
records evidence. Connected mode can ingest the same evidence so security teams
can audit rotation history across repos.

## 4. Org-wide model/method audit

Security asks: "Which workflows reference `app-deployer.apply` across all
teams?" Connected provenance history answers this by `change_id`, repo, and
policy outcome.

## 5. DRY-to-LIVE visibility for Swamp

For Swamp workflows, the important trace is not template field expansion; it is
"which workflow step caused which live mutation." This example focuses on
structural workflow governance and sets up that runtime-provenance direction.
