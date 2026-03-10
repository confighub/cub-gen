# ConfigHub Actions — User Stories

## 1. New commit triggers lifecycle

A developer pushes a feature branch merge to main. The commit trigger fires the ConfigHub Actions lifecycle: first a dry-run shows what would change, then a policy check verifies the changes pass platform rules, then the governed apply executes. `cub-gen` records the full provenance chain linking the commit to the lifecycle execution.

## 2. Production approval gate

The deploy action for production requires two approvals. When the lifecycle reaches the verify step, it pauses and notifies the platform-owner and security-lead. Both approve. The decision engine records both approvals with timestamps and reasons, creating an auditable approval chain.

## 3. Security tightens approval requirements

The security team increases required approvals from 2 to 3 in `operations-prod.yaml`. This change to the lifecycle definition itself goes through the same lifecycle — plan → verify → deploy. The recursive governance ensures that even governance policy changes have full provenance.

## 4. Deploy window enforcement

A developer tries to push a production release at 11 PM UTC (outside business hours). The lifecycle policy blocks the deploy step with a clear message: "production deploys restricted to Mon-Fri 09:00-17:00 UTC." The change is queued until the next business-hours window, with the full decision trail preserved.
