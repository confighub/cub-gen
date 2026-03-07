# New Inputs and Feedback Intake

## Newly Added Inputs

1. `../06-blog/04-agentic-gitops-blog-v2-feedback.md`
2. `../07-external-input/01-iits-whitepaper-contribution-agentic-gitops.md`
3. `../07-external-input/02-weave-whitepaper-gitops-enterprise-ai-enabled-platforms.pdf`

## Suggested Integration Pass

1. Normalize terminology to one control-loop model:
   `import -> propose -> evaluate -> approve -> execute -> attest -> promote`.
2. Keep storage boundary explicit in every external-facing narrative:
   `Git/OCI as transport + collaboration; ConfigHub as governance and WET authority`.
3. Ensure LIVE-origin lane is explicit in narrative docs:
   `LIVE -> CH MR proposal`, never silent overwrite.
4. Keep authority split explicit:
   source merge approval vs deploy decision approval.
5. Route reusable app changes into upstream platform DRY via promotion PR/MR.

## Next Optional Step

Create a single "Blog v3" draft that incorporates the feedback deltas and links back to:

1. `../02-design/00-agentic-gitops-design.md`
2. `../02-design/50-dual-approval-gitops-gh-pr-and-ch-mr.md`
3. `../03-worked-examples/01-scoredev-dry-wet-unit-worked-example.md`
4. `../03-worked-examples/02-traefik-helm-dry-wet-unit-worked-example.md`
