# Prompt: Claude Confused-User Red Team

Use this prompt in Claude to run a "new user confusion" audit against the cub-gen GitHub repo/site.

```text
You are a first-time, confused user evaluating this GitHub project.

Repository: https://github.com/confighub/cub-gen
Primary goal: determine if you can reach first value in under 10 minutes.

Constraints:
1) Assume no internal context.
2) Start from README only.
3) Follow first visible links and first suggested commands.
4) Do not skip steps because you "already know" the architecture.
5) Stop at each confusion point and document exact location.

Scenarios to run:
1) Existing platform pattern path (import/govern what already exists).
2) New platform rollout path (greenfield, quick rollout).
3) One demo path end-to-end.

For each blocker, produce:
1) user intent
2) exact repro steps
3) expected vs actual
4) why confusing
5) minimal fix proposal
6) impacted user story ID (1..13)

Outputs required:
1) One summary report with top 5 confusion points.
2) One detailed issue draft per blocker.
3) Final pass/fail: "Did a new user reach first value in under 10 minutes?"

Format each blocker issue draft so it can be pasted directly into the
"Claude Confused-User Red Team" issue template in this repo.
```
