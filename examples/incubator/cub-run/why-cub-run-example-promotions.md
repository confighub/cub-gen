# Why `cub run`: Evidence from `promotion-demo-data`

This document captures observations from running the `promotion-demo-data` example and how they validate the `cub run` PRD.

## What the Example Does

The `promotion-demo-data` example creates a realistic multi-app, multi-environment dataset in ConfigHub. It runs a shell script with 7 distinct phases:

| Phase | Action | Count |
|-------|--------|-------|
| **1: Infrastructure** | Created shared worker space + target spaces (one per cluster) | 1 worker + 7 targets |
| **2: App spaces** | Created deployment spaces for each app × target combination | 42 spaces |
| **3: Units in dev** | Created units from YAML templates in `us-dev-1` only | 22 units |
| **4a: Clone to lower** | Cloned dev units → dev-2, qa, us-staging, eu-staging | 24 clones |
| **4b: Clone to prod** | Cloned staging → prod (US→US, EU→EU) | 12 clones |
| **4c: Labeling** | Applied App/Owner/Role/Region labels to all units | 42 spaces |
| **5: Customize** | Set namespace, hostname, replicas, env vars per environment | 42 spaces |
| **6: Prod scale** | Increased CPU/memory requests/limits for prod | 12 spaces |
| **6b: Version skew** | Intentional `:4.2.0` vs `:4.2.1` in us-prod-1-eshop | 1 space |

**Totals**: 49 spaces, ~154 units, ~300 cub CLI invocations.

## Why This Validates `cub run`

This example is a near-perfect `cub run` candidate.

### 1. Clear multi-phase structure that should be steps

The script has 7 distinct phases. These map directly to what `cub run` proposes:

```
Step 1/7: infrastructure       DONE
Step 2/7: app-spaces           DONE
Step 3/7: units-in-dev         DONE
Step 4/7: clone-lower-envs     DONE
Step 5/7: clone-prod           DONE
Step 6/7: labeling             DONE
Step 7/7: customize            DONE
```

The phases have ordering constraints — you cannot label units before they exist, you cannot clone to prod before staging exists.

### 2. No assertions — biggest gap

The script prints "Done" after each phase but **never validates outcomes**.

The PRD's `Assertion` concept would add:

```
Assertion: all 49 spaces created     PASS
Assertion: all units labeled         PASS
Assertion: prod CPU >= 500m          PASS
Assertion: version skew exists       PASS
```

Today, if labeling silently fails on 3 spaces, the script still prints "Done."

### 3. No operational record

If this script fails at Phase 5, there's no ConfigHub record of what completed.

`cub run` would persist:

- which steps passed
- which failed
- where to resume
- what was actually created

A human or AI resuming after interruption would know exactly where to continue.

### 4. Silent error suppression is dangerous

Phase 5 (`Customizing per-environment`) uses this pattern extensively:

```bash
$CUB function do set-namespace "$space" --space "$space" --quiet 2>/dev/null || true
$CUB function do set-hostname "${space}.demo.confighub.io" --space "$space" --quiet 2>/dev/null || true
```

Every mutation swallows errors. If `set-replicas` fails for all prod spaces, the script still succeeds.

With `cub run --assert`, failures would be explicit instead of swallowed.

### 5. This is a "procedure" not a "command"

The PRD calls out exactly this: multi-step bounded procedures shouldn't be shell scripts.

This demo is:

- **7 phases with ordering constraints**
- **~300 cub invocations**
- **no shared state between phases except ConfigHub itself**

That's exactly the shape where ConfigHub should own the operational record.

## Suggested `cub run` Profile

If `cub run` existed today, this example would become:

```bash
cub run demo-data/install --record summary --assert
```

Which would produce:

```
Operation: op-01JDEMO...
Procedure: demo-data/install
Subject: demo-data

Step 1/7: infrastructure        DONE
Step 2/7: app-spaces            DONE
Step 3/7: units-in-dev          DONE
Step 4/7: clone-lower-envs      DONE
Step 5/7: clone-prod            DONE
Step 6/7: labeling              DONE
Step 7/7: customize             DONE

Assertion: 49 spaces created    PASS
Assertion: 154 units created    PASS
Assertion: all units labeled    PASS
Assertion: prod scaled          PASS
Assertion: version skew exists  PASS

Procedure state: done
Done: yes
```

## What This Means for MVP

1. **`demo-data/install` is a strong candidate for the second procedure profile** (after `global-app/install`). It's simpler because it doesn't require a live target — it uses the noop bridge.

2. **The assertion set is clear**:
   - count spaces by label
   - count units by label
   - verify label presence on units
   - verify resource values on prod units
   - verify image version skew exists

3. **The step structure is already defined** by the script phases. The CLI would just need to wrap them.

## Comparison to PRD Examples

The PRD mentions `realistic-app` as evidence. `promotion-demo-data` is arguably stronger evidence because:

| Aspect | `realistic-app` | `promotion-demo-data` |
|--------|-----------------|----------------------|
| Requires live target | Yes | No (noop bridge) |
| Number of spaces | ~10 | 49 |
| Number of units | ~20 | ~154 |
| Clone chains | Yes | Yes, multi-level |
| Verification script | Yes | No (gap to fix) |
| Error handling | Some | Suppressed |

Both examples show the need. `promotion-demo-data` shows it at scale.

## Related Files

- PRD: [03-cub-run-prd.md](./03-cub-run-prd.md)
- RFC: [03-cub-run-rfc.md](./03-cub-run-rfc.md)
- Example: [`confighub/examples/promotion-demo-data`](https://github.com/confighub/examples/tree/main/promotion-demo-data)
