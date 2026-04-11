# Prompts: `helm-paas`

Copy one block into Claude, Codex, or Cursor when you want an assistant to run
this example safely from the repo root.

## Safe preview prompt

```text
You are helping me inspect ./examples/helm-paas in the cub-gen repo.

Start in read-only mode.

1. Build ./cub-gen only if it is missing.
2. Run:
   ./cub-gen gitops import --space platform --json ./examples/helm-paas | jq '{
     profile: .discovered[0].generator_profile,
     dry_inputs,
     wet_manifest_targets,
     inverse_hint: .provenance[0].inverse_edit_pointers[0]
   }'
3. Summarize:
   - generator profile
   - main DRY inputs
   - one rendered target
   - one inverse edit hint with owner and confidence
   - whether `values-prod.yaml` or `values.yaml` is the winning source for the rendered image tag
4. Tell me exactly what the command read and what it did not mutate.

Do not run connected or live commands yet.
Stop if any prerequisite is missing.
```

## Local governed proof prompt

```text
Run the local governed ownership proof for ./examples/helm-paas.

1. Use:
   ./examples/helm-paas/demo-governed-change.sh
2. Capture the artifact directory from the script output.
3. Summarize:
   - which app-team edit passed
   - which platform-owned edit failed
   - the key files created under the artifact directory
4. Show me the most important evidence from:
   - allow-explain.json
   - allow-report.json
   - block-report.json

Do not run ConfigHub or live cluster steps in this prompt.
```

## Connected plus live prompt

```text
Help me run the connected and optional live Helm proof for ./examples/helm-paas.

Safety rules:
- Run the shared connected smoke check before the example-specific connected flow.
- Stop immediately if cub auth/context is missing.
- Do not run the live runtime path unless I explicitly asked for cluster mutation.

Sequence:
1. cub auth login
2. ./examples/demo/run-connected-smoke.sh
3. OUTPUT_DIR=.tmp/helm-paas-connected ./examples/helm-paas/demo-connected.sh
4. Only if I confirmed the live step:
   RECONCILER=both ./examples/helm-paas/demo-runtime.sh

After each step, tell me:
- what changed
- what stayed local/backend/live
- what to inspect next in ConfigHub or the cluster
```
