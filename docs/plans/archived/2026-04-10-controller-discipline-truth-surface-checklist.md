---
template_version: 0.2.0
created_at: "2026-04-10T08:45:00+08:00"
source_type: direct_request
source_refs: []
---

# Controller discipline truth-surface checklist

## Goal

Codify the controller-side lessons from the delta-review anchor slice so strong
agents miss fewer pre-existing problems and make fewer stateful or remote
operation mistakes during execution. The accepted direction is a two-layer
design: keep a small set of stable controller rules in `harness-execute`, then
add one lightweight phase-based checklist artifact the controller consults at
high-risk transitions instead of bloating the skill into a long playbook.

This slice should make controller self-check moments explicit across the whole
execution loop. The checklist is for controllers, not reviewers. Reviewer
behavior should continue to live primarily in `harness-reviewer`, while the new
controller artifact focuses on truth surfaces that were easy to skip in the
recent retrospective: review completeness, submission/round truth, archive
closeout truth, and PR/CI/sync truth.

## Scope

### In Scope

- Define a two-layer controller discipline model:
  - stable controller defaults in `harness-execute`
  - one lightweight controller checklist artifact for high-risk transitions
- Add explicit controller self-check guidance for four phases:
  - `pre-review`
  - `pre-aggregate`
  - `pre-archive`
  - `pre-land`
- Keep the checklist focused on the two failure classes accepted in discovery:
  - `漏看型`: pre-existing issues not caught at the right self-check/review moment
  - `误操作型`: lock collisions, CI misreads, stale remote assumptions, or noisy run handling
- Clarify which controller checks are stable defaults that belong in the skill
  contract versus which are runtime self-check prompts that belong in the
  checklist artifact.
- Sync bootstrap outputs so the repo-local `.agents` copy matches the source
  controller discipline guidance.

### Out of Scope

- Adding a separate reviewer checklist
- Turning `harness-execute` into a long exhaustive SOP or workflow engine
- Reworking `harness-reviewer` beyond any minimal cross-reference needed for
  controller/reviewer alignment
- Changing review/state/archive semantics that already landed in the previous
  delta-review anchor slice
- Adding CLI-enforced hard gates for every checklist item

## Acceptance Criteria

- [x] The resulting design clearly separates stable controller defaults from the
      lightweight controller checklist, so a cold reader can see what belongs
      in `harness-execute` versus the checklist artifact.
- [x] `harness-execute` guidance adds strong-default controller self-check
      moments without turning the skill into a long operational wall of text.
- [x] The controller checklist is organized by the four accepted phases:
      `pre-review`, `pre-aggregate`, `pre-archive`, and `pre-land`.
- [x] Each phase stays lean, with only the highest-signal truth-surface checks
      rather than a long compliance-style list.
- [x] `pre-review` guidance explicitly covers scope truth, anchor/diff truth,
      contract scan, and dispatch sanity.
- [x] `pre-aggregate` guidance explicitly covers submission truth, round-state
      truth, and a light synthesis sanity check before aggregation.
- [x] `pre-archive` guidance explicitly covers placeholder debt and narrative
      debt before the controller archives a candidate.
- [x] `pre-land` guidance explicitly covers PR truth, CI truth, sync truth, and
      merge/bookkeeping truth before or during land.
- [x] Reviewer behavior remains primarily owned by `harness-reviewer`; the new
      artifact is clearly controller-primary rather than a second reviewer
      protocol.
- [x] `scripts/sync-bootstrap-assets` refreshes the materialized `.agents`
      outputs and `scripts/sync-bootstrap-assets --check` passes afterward.

## Deferred Items

- Consider later whether any checklist item has proved important enough in
  practice to justify a CLI-enforced hard gate instead of skill-level guidance.

## Work Breakdown

### Step 1: Define the two-layer controller discipline contract

- Done: [x]

#### Objective

Decide exactly which controller truths belong in the stable `harness-execute`
 contract and which belong in the lightweight checklist artifact.

#### Details

Use the accepted discovery framing rather than reopening it during execution:

- the checklist is controller-primary, not shared equally with reviewers
- the main goal is to reduce both completeness misses and operation mistakes
- the checklist should be phase-based, with risk-focused bullets inside each
  phase
- language strength should be strong-default guidance, not soft suggestions and
  not a hard gate engine

This step should leave behind a crisp split that future agents can reuse:
`harness-execute` owns the stable defaults and when to pause for self-checks;
the checklist artifact owns the concise per-phase scan content.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/`

#### Validation

- A cold controller agent could explain the difference between the stable skill
  contract and the lightweight checklist artifact without relying on discovery
  chat.
- The resulting split stays lean enough that `AGENTS.md` does not need to grow
  and `harness-execute` does not become an exhaustive SOP.

#### Execution Notes

Defined the split in the bootstrap source instead of expanding the managed
`AGENTS.md` block: `assets/bootstrap/skills/harness-execute/SKILL.md` now owns
the stable controller defaults and points cold controllers at a dedicated
truth-surface checklist reference, while the phase-by-phase scan content lives
outside the core skill body. Updated the execute references that actually own
review, archive, and publish transitions so the new self-check moments are
discoverable where controllers already look.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 and Step 2 landed as one tightly coupled
guidance slice, so a separate step-bound review here would have duplicated the
later candidate-level finalize review.

### Step 2: Add the phase-based truth-surface checklist

- Done: [x]

#### Objective

Create the lightweight controller checklist artifact that covers the four
accepted high-risk phases.

#### Details

The checklist should be phase-based and intentionally short. Each phase should
carry only the checks that matter most:

- `pre-review`
  - scope truth
  - anchor/diff truth
  - contract scan
  - dispatch sanity
- `pre-aggregate`
  - submission truth
  - round-state truth
  - synthesis sanity
- `pre-archive`
  - placeholder debt
  - narrative debt
  - light publish-readiness confirmation
- `pre-land`
  - PR truth
  - CI truth
  - sync truth
  - merge/bookkeeping truth

Write it so another strong agent would actually use it during execution instead
of skimming past it as filler. Keep it decisional and concrete rather than
turning it into a generic essay about being careful.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/references/`
- `.agents/skills/harness-execute/references/`

#### Validation

- Each phase fits on a short scan and clearly targets one of the two accepted
  failure classes from discovery.
- The checklist is controller-primary and does not silently morph into a second
  reviewer protocol.

#### Execution Notes

Added `controller-truth-surfaces.md` under the bootstrap execute references and
kept each phase intentionally short: `pre-review`, `pre-aggregate`,
`pre-archive`, and `pre-land` each now carry only the accepted truth-surface
checks from discovery. The checklist is written as controller-facing decisional
prompts, not as reviewer instructions or a compliance wall.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The checklist content is inseparable from Step 1's
contract split, so the final packaged candidate is reviewed as one bounded
controller-discipline change.

### Step 3: Sync the controller workflow guidance and validate the packaged result

- Done: [x]

#### Objective

Materialize the new controller discipline guidance into the dogfood outputs and
prove the packaged guidance remains in sync.

#### Details

After editing the bootstrap source files, sync the bootstrap outputs so the
rooted `.agents` copy reflects the new controller guidance. Validate the final
result with the bootstrap sync checks and any focused tests needed if the
controller references or packaging logic require updates.

Keep execution detail concise. The point of this step is not to add runtime
behavior changes; it is to make sure the packaged guidance a future agent sees
actually matches the source and the accepted discovery direction.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/`
- `assets/bootstrap/skills/harness-land/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/`
- `.agents/skills/harness-land/SKILL.md`

#### Validation

- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-04-10-controller-discipline-truth-surface-checklist.md`

#### Execution Notes

Ran `scripts/sync-bootstrap-assets` to materialize the updated execute and land
guidance into `.agents/`, then verified the package remained clean with
`scripts/sync-bootstrap-assets --check` and `harness plan lint
docs/plans/active/2026-04-10-controller-discipline-truth-surface-checklist.md`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 only materialized and validated the same
controller-guidance slice, so separate step review would add little beyond the
finalize review of the packaged result.

## Validation Strategy

- Keep the validation centered on guidance quality and packaged sync fidelity,
  not on introducing new runtime mechanics unless execution genuinely requires
  them.
- Re-read the resulting skill plus checklist as a cold controller agent and
  verify that the four high-risk transitions each have a concise truth-surface
  scan.
- Use bootstrap sync checks to ensure the repo-local `.agents` materialization
  stays aligned with the edited bootstrap source.

## Risks

- Risk: The controller checklist could grow into another long SOP that agents
  skim instead of use.
  - Mitigation: Keep each phase lean and focused on a few truth-surface checks
    rather than exhaustive prose.
- Risk: Stable defaults and checklist prompts could duplicate each other and
  blur the intended two-layer split.
  - Mitigation: Decide the split explicitly in Step 1 before filling in the
    checklist details.
- Risk: Reviewer behavior could accidentally get redefined in controller docs,
  causing drift between `harness-execute` and `harness-reviewer`.
  - Mitigation: Keep the checklist controller-primary and leave reviewer
    behavior in `harness-reviewer` except for minimal alignment references.

## Validation Summary

- Re-read the edited bootstrap and materialized `.agents` guidance as a cold
  controller path and confirmed the stable-default vs checklist split stays in
  `harness-execute` while phase-specific scans stay in
  `controller-truth-surfaces.md`.
- Ran `scripts/sync-bootstrap-assets` after the initial implementation and
  after both finalize-review follow-up fixes so the packaged `.agents` output
  stayed aligned with `assets/bootstrap/`.
- Verified packaged sync with `scripts/sync-bootstrap-assets --check`.
- Verified the tracked plan remained valid with `harness plan lint
  docs/plans/active/2026-04-10-controller-discipline-truth-surface-checklist.md`.
- After reopening for revision 2 because `origin/main` advanced, merged
  `origin/main` into the branch and reran focused validation with `go test
  ./internal/lifecycle ./internal/plan ./internal/status`.

## Review Summary

- `review-001-full` found one important workflow-semantics issue: the new
  `Pre-Land` scan was wired before the archived candidate had publish/CI/sync
  truth available. The fix moved that scan later in the archived-candidate
  handoff, after publish, CI, and sync evidence exist.
- `review-002-full` found one important docs-consistency issue: the archive
  handoff still implied `execution/finalize/await_merge` could appear before
  publish/CI/sync evidence. The fix rewrote `closeout-and-archive.md` so the
  post-archive flow explicitly reaches `execution/finalize/publish` first, then
  evidence submission, then a later `await_merge` status check.
- `review-003-full` passed clean with no blocking or non-blocking findings
  after those two follow-up fixes.
- Revision 2 reopened the archived candidate after `origin/main` advanced by
  four commits and overlapped this slice's bootstrap and plan files.
  `review-004-full` then found three blocking drift issues: stale revision-1
  archive facts still left in the reopened active plan, a runtime archive next
  action that still permitted merge before explicit human approval, and a small
  Step 3 expected-files mismatch around the touched `harness-land` files.
- `review-005-full` found two remaining revision-2 correctness drifts: the
  reopened active plan still exposed archive-era facts as current state, and
  the archive runtime next actions still skipped the publish-phase
  publish/CI/sync evidence handoff.
- `review-006-full` passed clean with no blocking or non-blocking findings
  after those revision-2 plan and runtime truth-surface fixes.

## Archive Summary

- Archived At: 2026-04-10T09:36:35+08:00
- Revision: 2
- Reopen History: Revision 1 was archived at `2026-04-10T09:19:51+08:00`, then
  invalidated by remote drift from `origin/main` and reopened into revision 2.
- PR: [#129](https://github.com/catu-ai/easyharness/pull/129)
- Ready: Revision 2 now has a clean finalize full review after merging
  `origin/main`, refreshing the active plan truth surfaces, tightening the
  archive runtime next actions, and rerunning focused validation plus bootstrap
  sync checks.
- Merge Handoff: Commit and push the refreshed revision-2 archive plus
  runtime/test updates, update PR #129, then record fresh publish, CI, and
  sync evidence before waiting for explicit human merge approval.

## Outcome Summary

### Delivered

- Added a dedicated controller-only `controller-truth-surfaces.md` reference
  that keeps `pre-review`, `pre-aggregate`, `pre-archive`, and `pre-land`
  checks concise and phase-based instead of bloating `harness-execute`.
- Updated `harness-execute` bootstrap guidance to add strong-default
  controller self-check moments and linked those moments from the review,
  archive, and publish handoff references.
- Updated `harness-land` to re-read the `Pre-Land` scan at merge time so land
  still refreshes PR, CI, sync, and bookkeeping truth before merge-sensitive
  work.
- Synced the packaged `.agents` outputs and iterated through three finalize
  review rounds until the controller-discipline story was internally
  consistent.
- Reopened the candidate for revision 2 after remote drift from `origin/main`,
  merged the latest mainline changes, and completed the runtime and plan
  truth-surface refresh needed for a clean re-archive.

### Not Delivered

- [#128](https://github.com/catu-ai/easyharness/issues/128): Decide later which
  controller truth-surface checks have proved important enough to promote from
  skill guidance to CLI hard gates.

### Follow-Up Issues

- [#128](https://github.com/catu-ai/easyharness/issues/128): Track whether any
  controller truth-surface checks should graduate from workflow guidance to
  CLI-enforced hard gates.
