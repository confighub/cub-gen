# Triple Style Comparison

This folder provides three fully populated representations of the generator triple model for all supported generator kinds.

1. Style A (`style-a-yaml`): YAML-first representation.
2. Style B (`style-b-markdown`): human-readable markdown + tables.
3. Style C (`style-c-yaml-plus-docs`): YAML + markdown pair per kind.

Canonical runtime source remains Go registry specs in `internal/registry/registry.go`.

| Kind | Style A | Style B | Style C |
| --- | --- | --- | --- |
| `ably` | [yaml](style-a-yaml/ably.yaml) | [markdown](style-b-markdown/ably.md) | [pair](style-c-yaml-plus-docs/ably/) |
| `backstage` | [yaml](style-a-yaml/backstage.yaml) | [markdown](style-b-markdown/backstage.md) | [pair](style-c-yaml-plus-docs/backstage/) |
| `c3agent` | [yaml](style-a-yaml/c3agent.yaml) | [markdown](style-b-markdown/c3agent.md) | [pair](style-c-yaml-plus-docs/c3agent/) |
| `helm` | [yaml](style-a-yaml/helm.yaml) | [markdown](style-b-markdown/helm.md) | [pair](style-c-yaml-plus-docs/helm/) |
| `opsworkflow` | [yaml](style-a-yaml/opsworkflow.yaml) | [markdown](style-b-markdown/opsworkflow.md) | [pair](style-c-yaml-plus-docs/opsworkflow/) |
| `score` | [yaml](style-a-yaml/score.yaml) | [markdown](style-b-markdown/score.md) | [pair](style-c-yaml-plus-docs/score/) |
| `springboot` | [yaml](style-a-yaml/springboot.yaml) | [markdown](style-b-markdown/springboot.md) | [pair](style-c-yaml-plus-docs/springboot/) |
| `swamp` | [yaml](style-a-yaml/swamp.yaml) | [markdown](style-b-markdown/swamp.md) | [pair](style-c-yaml-plus-docs/swamp/) |

## Regenerate

```bash
go run ./cmd/cub-gen-style-sync
```
