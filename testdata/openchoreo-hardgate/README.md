# OpenChoreo Hard Gate Fixture

This fixture is the credibility test for OpenChoreo-style platform support.
It is intentionally small, but it includes the hard parts:

- app-owned `Workload`
- platform-owned `ComponentType`
- environment-owned `ReleaseBinding` for `dev` and `prod`
- security-owned `SecretReference`
- controller-owned `RenderedRelease`
- generated Kubernetes `Deployment`, `Service`, and mounted-file `ConfigMap`
  resources

Run:

```bash
./cub-gen gitops discover --space platform --adoption-report --json testdata/openchoreo-hardgate
./cub-gen gitops import --space platform --json testdata/openchoreo-hardgate
```

This proves fixture-backed initial support. It is not a claim of full upstream
OpenChoreo conformance.
