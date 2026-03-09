# Plan: New-User Challenges And Confused-User Red Team

**Status**: Proposed  
**Date**: 2026-03-09  
**Owner**: DX + Product + Platform Engineering

## Goal

Convert "new user confusion" into fast, measurable product and docs improvements.

## Intake model (GitHub issues)

Use a dedicated issue class for first-run blockers and misunderstanding points.

Required fields:

1. user goal
2. expected outcome
3. actual outcome
4. exact command/path
5. where they got stuck
6. what they tried

Label taxonomy:

1. `challenge:new-user`
2. `area:docs|cli|demo|adapter|bridge`
3. `severity:blocking|friction`
4. `story:1..13`

## Triage workflow

1. Acknowledge within 1 business day.
2. Reproduce in same environment path.
3. Classify root cause:
   - messaging gap
   - missing guardrail
   - command UX issue
   - missing capability
4. Apply fix path:
   - docs-only PR
   - CLI/demo patch PR
   - product gap issue linked to roadmap
5. Close only when:
   - reproduction no longer fails, and
   - updated docs/demo path exists.

## Service levels

1. First response SLA: 1 business day.
2. Repro SLA: 3 business days.
3. Fix ETA:
   - docs/UX fix: same sprint
   - capability gap: scheduled with milestone and owner

## Weekly challenge review

1. Top repeated blockers by frequency.
2. Median time-to-first-value from newcomer reports.
3. Reopen rate by area.
4. New issue -> merged fix lead time.

## Claude confused-user red team

Run weekly as a "new, confused user" against the GitHub repo/site.

Mission:

1. Attempt onboarding from README only.
2. Attempt one "existing platform" path.
3. Attempt one "new rollout" path.
4. File issues for every confusion point with exact repro.

Rules:

1. Assume no internal context.
2. Do not infer hidden architecture.
3. Prefer first visible path and first visible command.
4. Stop at first blocker and file issue immediately.

Expected output per run:

1. One summary issue: top 5 confusion points.
2. One issue per blocker with:
   - user intent
   - exact repro steps
   - expected vs actual
   - suggested minimal fix
3. A pass/fail score for first-run success in under 10 minutes.

## Definition of done

1. New-user issues are triaged with labels and linked stories.
2. Weekly red-team run is completed and tracked.
3. Repeated confusion themes decline over time.
4. Time-to-first-value trend improves release over release.
