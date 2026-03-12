# Domain POV Matrix

This matrix captures how different domain users evaluate `cub-gen` and which
message should lead each example.

## 1. Spring Boot shops

- Primary user: app teams + platform DBA/ops boundaries.
- First pain: "Which Spring property changed, and who owns it?"
- Best first value: ALLOW/BLOCK feedback in Spring property terms.
- Avoid leading with: Kubernetes internals and long bridge mechanics.

## 2. Helm platform teams (IITS-style environments)

- Primary user: platform consultants/engineers with umbrella charts, overlays,
  and Flux/Argo transport layers.
- First pain: value precedence and ownership archaeology during incidents.
- Best first value: inventory + ownership map + inverse edit path.
- Avoid leading with: full attestation pipeline before visibility is proven.

## 3. Score.dev platform teams

- Primary user: platform team maintaining Score render/provisioning contracts.
- First pain: no clear trace from `score.yaml` intent to runtime fields.
- Best first value: post-render provenance + ownership boundaries.
- Avoid leading with: generic Kubernetes authoring concepts (developers do not
  author Kubernetes directly).

## 4. Swamp / workflow-native teams

- Primary user: workflow maintainers and security/compliance leads.
- First pain: structural workflow mutation risk (models/methods/required steps).
- Best first value: workflow change classification and policy-ready metadata.
- Avoid leading with: Helm-style DRY->WET rendering assumptions.

## 5. Ops workflow teams

- Primary user: SRE/operations automation owners.
- First pain: schedule/action changes are high impact but weakly governed.
- Best first value: explicit ownership and decision state on workflow changes.
- Avoid leading with: app deployment narratives not relevant to operations flow.

## 6. Backstage catalog owners

- Primary user: IDP/catalog admins and service owners.
- First pain: ownership/lifecycle churn during reorg/compliance updates.
- Best first value: catalog change provenance and policy checks by field.
- Avoid leading with: cluster reconciliation details.

## 7. AI fleet/platform teams

- Primary user: AI platform owners balancing speed and safety.
- First pain: model/budget/credential changes fan out quickly.
- Best first value: short fleet intent with traceable, governed runtime output.
- Avoid leading with: tool replacement narratives (Flux/Argo remain).

## Cross-domain messaging rules

- Lead with existing workflow compatibility.
- Explain "what changes for me tomorrow morning" before architecture terms.
- Keep WET details primarily for platform/operator readers.
- Treat verification + attestation as trust boundary, not as optional garnish.
