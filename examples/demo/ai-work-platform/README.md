# AI work platform demo track

These demos simulate AI-work platform use cases while keeping the same local-first `cub-gen` import model.

## Domain POV

This track is for teams combining:

- app or prompt-level intent authored by humans/agents,
- workflow-style execution models (Swamp/ops actions),
- Kubernetes GitOps runtime (Flux/Argo) where needed.

Use this track to validate the closed loop:
intent -> governed change bundle -> decision -> runtime outcome -> next edit guidance.

Scenarios:

1. C3 Agent (`c3agent`)
   - Demonstrates 11-target manifest-set metadata coverage with inverse ownership routing
   `./examples/demo/ai-work-platform/scenario-1-c3agent.sh`

2. Swamp Automation (`swamp`)
   `./examples/demo/ai-work-platform/scenario-2-swamp.sh`

3. ConfigHub Actions (`ops-workflow`)  
   `./examples/demo/ai-work-platform/scenario-3-confighub-actions.sh`

4. Operations workflow (`ops-workflow`)  
   `./examples/demo/ai-work-platform/scenario-4-operations.sh`

Run all:

`./examples/demo/ai-work-platform/run-all.sh`
