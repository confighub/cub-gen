# Prompts: `scoredev-paas`

Copy one block into Claude, Codex, or Cursor when you want an assistant to run
this example safely from the repo root.

## Safe preview prompt

```text
You are helping me inspect ./examples/scoredev-paas in the cub-gen repo.

Start read-only.

1. Build ./cub-gen only if needed.
2. Run:
   ./cub-gen gitops import --space platform --json ./examples/scoredev-paas | jq '{
     profile: .discovered[0].generator_profile,
     dry_inputs,
     wet_manifest_targets,
     inverse_hint: .provenance[0].inverse_edit_pointers[0]
   }'
3. Run:
   ./cub-gen score validate-workload \
     --score ./examples/scoredev-paas/score.yaml \
     --contract ./examples/scoredev-paas/platform/contracts/workload-class.yaml
4. Summarize the provenance result and whether the current workload is allowed.

Do not run connected flows yet.
Tell me exactly what was read and what stayed unmodified.
```

## Local governed proof prompt

```text
Run the local governed Score proof for ./examples/scoredev-paas.

1. Use:
   ./examples/scoredev-paas/demo-governed-workload.sh
2. Capture the artifact directory from the script output.
3. Summarize:
   - which app-owned change stayed allowed
   - which resource addition escalated
   - which files in the artifact directory prove each outcome

Do not run backend or live-cluster steps in this prompt.
```

## Connected prompt

```text
Help me run the connected Score walkthrough for ./examples/scoredev-paas.

Safety rules:
- Start with the shared connected smoke check even though Score is not in the smoke lane.
- Stop if cub auth/context is missing.
- Be explicit that this example still does not claim standalone live-cluster Score proof.

Sequence:
1. cub auth login
2. ./examples/demo/run-connected-smoke.sh
3. OUTPUT_DIR=.tmp/scoredev-connected ./examples/scoredev-paas/demo-connected.sh

After each step, tell me what to inspect in ConfigHub and what the remaining proof gap is.
```
