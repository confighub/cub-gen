# App-of-Apps Generator Boundary

`cub-gen` models Argo app-of-apps as a generator family when a repo has a root
Argo `Application` whose `spec.source.path` points at a local child
`Application` catalog.

The supported bounded shape is:

```text
root Application
  -> child Application catalog
      -> downstream Helm/Kustomize/plain manifest source
```

## What Is Supported

1. **Root detection**
   The repo has an Argo `Application` such as `root-application.yaml` or
   `app-of-apps.yaml`.

2. **Local child catalog**
   The root `spec.source.path` resolves to a local directory such as `apps/`.
   Child YAML files in that directory are parsed as Argo `Application`
   resources.

3. **Provenance**
   Import emits root Application lineage for `spec.source.path`, child
   Application lineage for `metadata.name`, and child Application lineage for
   `spec.source.repoURL` and `spec.source.path`.

4. **Inverse edit routing**
   Child app edits route to the child app catalog. Root catalog path edits route
   to the root Application source.

## What Is Not Claimed Yet

- No remote repository crawling from a root app URL.
- No automatic downstream Helm/Kustomize import unless that downstream source is
  also present as an explicit generator repo/input.
- No attempt to infer child apps when the root source path is absent or remote
  only.

Unknown or unsupported shapes are not guessed.

## Proof

Fixture:

```bash
./cub-gen gitops discover --space platform ./testdata/app-of-apps-standalone
./cub-gen gitops import --space platform --json ./testdata/app-of-apps-standalone \
  | jq '.provenance[0].app_of_apps_analysis'
```

Contract coverage:

```bash
go test ./internal/appofapps ./internal/detect ./internal/importer ./internal/contracts -count=1
go test ./cmd/cub-gen -run '^(TestGitOpsParityGoldenDiscoverAppOfApps|TestGitOpsParityGoldenImportAppOfApps)$' -count=1
```

