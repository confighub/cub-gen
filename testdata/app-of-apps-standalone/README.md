# App-of-Apps Standalone Fixture

This fixture proves that app-of-apps is treated as a generator boundary:

```text
root Application
  -> apps/ child Application catalog
      -> downstream Helm/Kustomize/plain manifest sources
```

The root Application selects `apps/`. The child Application YAML files are the
deterministic generated set for this bounded fixture.

