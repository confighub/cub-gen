# Prompts: `swamp-automation`

Copy one block into Claude, Codex, or Cursor when you want an assistant to run
this example safely from the repo root.

## Safe preview prompt

```text
You are helping me inspect ./examples/swamp-automation in the cub-gen repo.

Start in read-only mode.

1. Build ./cub-gen only if needed.
2. Run:
   ./cub-gen gitops import --space platform --json ./examples/swamp-automation | jq '{
     profile: .discovered[0].generator_profile,
     dry_inputs,
     wet_manifest_targets,
     inverse_hint: .provenance[0].inverse_edit_pointers[0]
   }'
3. Summarize:
   - generator profile
   - workflow DRY inputs
   - one governed output target
   - one inverse edit hint
4. Tell me exactly what was read and what stayed unmodified.

Do not run prompt-as-DRY or connected steps yet.
```

## AI-safe local prompt-as-DRY prompt

```text
Run the AI-safe local prompt-as-DRY loop for ./examples/swamp-automation.

1. Use:
   OUT_DIR=.tmp/swamp-prompt-local ./examples/demo/prompt-as-dry-local.sh ./examples/swamp-automation
2. Confirm the AI-only guardrails passed.
3. Summarize:
   - change_id
   - bundle_digest
   - attestation_digest
   - the top edit recommendation from mutation-card.json
4. Tell me which files were written under .tmp/swamp-prompt-local and whether anything touched repo, backend, or live state.

Do not run connected steps in this prompt.
```

## Local governed structure prompt

```text
Run the example-owned local governed proof for ./examples/swamp-automation.

1. Use:
   ./examples/swamp-automation/demo-governed-structure.sh
2. Capture the artifact directory from the script output.
3. Summarize:
   - which approved model-method change stayed ALLOW
   - which missing-step/unapproved-model change was BLOCKED
   - the key files created under the artifact directory
4. Show me the most important evidence from:
   - allow-summary.json
   - block-summary.json

Do not run connected or live steps in this prompt.
```

## Connected workflow prompt

```text
Help me run the connected workflow proof for ./examples/swamp-automation.

Safety rules:
- Start with the shared connected smoke check.
- Enforce the AI-only guardrails before the connected workflow run.
- Stop if cub auth/context is missing.
- Be explicit that this example proves structural governance more strongly than live runtime execution.

Sequence:
1. cub auth login
2. ./examples/demo/run-connected-smoke.sh
3. OUTPUT_DIR=.tmp/swamp-prompt-connected ./examples/demo/prompt-as-dry-connected.sh ./examples/swamp-automation
4. OUTPUT_DIR=.tmp/swamp-connected ./examples/swamp-automation/demo-connected.sh

After each step, tell me what to inspect in ConfigHub and which proof boundary is still partial.
```
