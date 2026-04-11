# Prompts: `springboot-paas`

Copy one block into Claude, Codex, or Cursor when you want an assistant to run
this example safely from the repo root.

## Safe preview prompt

```text
You are helping me inspect ./examples/springboot-paas in the cub-gen repo.

Stay read-only first.

1. Build ./cub-gen only if needed.
2. Run:
   ./examples/springboot-paas/generator/render.sh --explain
3. Run:
   ./cub-gen gitops import --space platform --json ./examples/springboot-paas | jq '{
     profile: .discovered[0].generator_profile,
     dry_inputs,
     wet_manifest_targets,
     inverse_hint: .provenance[0].inverse_edit_pointers[0]
   }'
4. Summarize the source boundary, one edit hint, and what the preview did not mutate.

Do not run connected or live-cluster steps yet.
```

## Local route and apply-here proof prompt

```text
Run the local Spring governance proofs for ./examples/springboot-paas.

1. Run:
   ./examples/springboot-paas/demo-governed-routes.sh
2. Then run:
   ./examples/springboot-paas/demo-embedded-config-mutation.sh
3. Capture both artifact directories.
4. Summarize:
   - which field is allowed
   - which field is blocked
   - how the embedded payload mutation proves the apply-here path
   - where the compare and refresh evidence lives

Do not run backend or live-cluster steps in this prompt.
```

## Connected plus live prompt

```text
Help me run the connected and optional live Spring proof for ./examples/springboot-paas.

Safety rules:
- Run the shared connected smoke check first.
- Stop immediately if cub auth/context is missing.
- Do not run the live-cluster commands unless I explicitly asked for cluster mutation.

Sequence:
1. cub auth login
2. ./examples/demo/run-connected-smoke.sh
3. OUTPUT_DIR=.tmp/springboot-connected ./examples/springboot-paas/demo-connected.sh
4. Only if I confirmed the live step:
   ./bin/create-cluster
   ./bin/build-image
   ./bin/install-worker
   ./examples/springboot-paas/verify-e2e.sh

After each step, tell me what to inspect in ConfigHub, the payload helpers, or the kind cluster.
```
