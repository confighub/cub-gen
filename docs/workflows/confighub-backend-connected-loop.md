# ConfigHub Backend-Connected Loop

This project (`cub-gen`) is a local-first parity harness for generator contracts and provenance.

If you have the ConfigHub backend available, run the backend-connected flow with `cub` CLI:

1. `cub gitops discover`
2. `cub gitops import`
3. `cub gitops cleanup`

This is the real server-connected path (spaces, units, mutations, targets), not the local simulation path.

## Preconditions

1. ConfigHub backend is running and reachable.
2. `cub` CLI is installed and authenticated.
3. A ConfigHub space exists with:
   - one discovery target (Kubernetes)
   - one renderer target (Flux/Argo renderer provider)

References:

1. ConfigHub repo: [github.com/confighubai/confighub](https://github.com/confighubai/confighub)
2. Backend + CLI local setup: [confighub README](https://github.com/confighubai/confighub/blob/main/README.md)
3. Full Argo import e2e walkthrough: [argocd-import-test-guide.md](https://github.com/confighubai/confighub/blob/main/docs/test/argocd-import-test-guide.md)

## Quick run

```bash
SPACE=<space-slug> \
DISCOVERY_TARGET=<discovery-target-slug> \
RENDER_TARGET=<renderer-target-slug> \
./examples/demo/confighub-connected-gitops.sh
```

Optional cleanup:

```bash
CLEANUP=1 \
SPACE=<space-slug> \
DISCOVERY_TARGET=<discovery-target-slug> \
RENDER_TARGET=<renderer-target-slug> \
./examples/demo/confighub-connected-gitops.sh
```

## Manual commands

```bash
cub target list --space "$SPACE"
cub gitops discover --space "$SPACE" "$DISCOVERY_TARGET"
cub gitops import --space "$SPACE" "$DISCOVERY_TARGET" "$RENDER_TARGET"
cub unit list --space "$SPACE"
cub gitops cleanup --space "$SPACE" "$DISCOVERY_TARGET"
```

## Scope clarity

`cub-gen` current bridge commands (`bridge ingest/decision`) remain local contract surfaces in this repo.
For backend-connected GitOps import and persisted units/mutations today, use the `cub` CLI flow above.
