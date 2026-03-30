---
template_version: 0.2.0
created_at: "2026-03-30T23:57:22+08:00"
source_type: issue
source_refs:
    - '#69'
---

# Add a lightweight workflow for low-risk changes

## Goal

Add an explicit lightweight workflow for narrow, low-risk repository changes so
humans can still steer through a plan, while agents avoid the full tracked-plan
ceremony for work such as tiny README or docs fixes.

The new path should stay inside the existing plan-driven model rather than
creating a second workflow object. A lightweight change still starts from a
plan, but that plan becomes command-owned local state instead of a tracked
repository artifact. The workflow must leave a small repo-visible breadcrumb so
reviewers can see that the lightweight path was used intentionally.

## Scope

### In Scope

- Define a lightweight workflow profile that keeps the existing plan schema and
  adds one optional explicit profile field rather than inventing a second
  object type.
- Specify which low-risk changes may use the lightweight path and which changes
  must stay on the standard tracked-plan path.
- Define where lightweight plans live, how they archive into `.local/`, and
  what minimum durable record remains repo-visible for reviewers.
- Extend `harness plan template` with a lightweight authoring mode that seeds a
  shorter single-step plan and low-risk closeout guidance.
- Update runtime behavior so status, archive, and related guidance understand
  lightweight plans and can remind agents to leave the agreed breadcrumb in PR
  bodies or similar review surfaces.
- Add tests and agent-facing guidance for when the lightweight profile is
  allowed and how to use it without bypassing human steering.

### Out of Scope

- Replacing the standard tracked-plan workflow for medium or large work.
- Treating all documentation changes as automatically lightweight-safe.
- Creating a second parallel lifecycle model that duplicates the existing plan,
  review, archive, and status concepts under a different artifact type.
- Building automatic GitHub PR body mutation into the CLI in this slice if
  guidance-only next actions are sufficient.

## Acceptance Criteria

- [x] The normative docs define an optional `workflow_profile` field and
      reserve at least `standard` and `lightweight` as explicit workflow
      choices under the same plan schema, with omitted values preserving the
      current behavior as `standard`.
- [x] The docs clearly define lightweight-path eligibility, including examples
      of acceptable low-risk changes and explicit reasons to stay on the
      standard tracked-plan path.
- [x] Lightweight plans are command-owned local artifacts rather than tracked
      files under `docs/plans/`, and lightweight archive records remain in
      `.local/harness/...` instead of moving into `docs/plans/archived/`.
- [x] `harness plan template` exposes a lightweight authoring mode that seeds a
      shorter low-risk plan shape without requiring a second template schema.
- [x] `harness status` and any relevant closeout commands provide explicit
      guidance that lightweight changes still need a repo-visible breadcrumb,
      such as a PR body note describing why the lightweight path was used.
- [x] Agent-facing docs explain that lightweight changes still require human
      steering through a plan, even though the plan and archive stay local.
- [x] Focused automated tests cover lightweight template generation plus at
      least one end-to-end lightweight flow through status and archive-local
      closeout behavior.

## Deferred Items

- Decide later whether lightweight changes need a dedicated CLI command for
  writing or validating repo-visible breadcrumbs beyond status guidance.
- Consider a future retrospective report or listing command for historical
  lightweight local archives if `.local/` discoverability proves too weak.

## Work Breakdown

### Step 1: Define the lightweight workflow contract

- Done: [x]

#### Objective

Write the durable product contract for lightweight plans, including the new
optional profile field, eligibility boundaries, local-only storage, and
breadcrumb expectations.

#### Details

Capture the accepted discovery decisions explicitly:
- plans still exist because humans need a steerable artifact
- lightweight plans do not belong in tracked `docs/plans/`
- the plan schema should evolve minimally, with one optional profile field
  instead of a second plan type
- omitted `workflow_profile` must preserve today's standard tracked-plan
  behavior
- lightweight archive records live in `.local/`
- agents still owe reviewers a repo-visible breadcrumb, likely via PR body
  wording or status guidance

The contract should be precise enough that future agents can tell when
lightweight is allowed and when a change must escalate back to `standard`.

#### Expected Files

- `docs/specs/plan-schema.md`
- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- `README.md`
- `AGENTS.md`

#### Validation

- The docs state the lightweight workflow in durable tracked files rather than
  leaving it in prompts or issue comments.
- A cold reader can tell where lightweight plans live, how they differ from
  standard plans, and what visible breadcrumb remains required.

#### Execution Notes

Updated `docs/specs/plan-schema.md`, `docs/specs/state-model.md`,
`docs/specs/state-transitions.md`, `docs/specs/cli-contract.md`, `README.md`,
and `AGENTS.md` to define the lightweight contract as an optional
`workflow_profile` with default `standard` behavior, local
`.local/harness/plans/<plan-stem>/active|archived/` storage for lightweight
plans, and an explicit repo-visible breadcrumb requirement. Reused the
existing node tree and workflow shape instead of introducing a second
lifecycle model. After review surfaced drifts around how lightweight plans
become current, updated the state-model and transition rules so they now agree
on three points: exactly one active plan must exist, the pointer can help
resume or disambiguate but cannot bypass that invariant, and a lone active
lightweight local plan remains recoverable when the pointer is missing or
stale. Reread the affected docs together to keep the standard-path defaults
and lightweight-path constraints aligned.

#### Review Notes

`review-001-delta` and `review-003-full` both requested changes before Step 1
could close. The repeated findings were all contract-alignment issues around
lightweight current-plan discovery: whether a missing or stale
`.local/harness/current-plan.json` pointer could fall back to exactly one
active lightweight local plan, and whether that pointer could bypass the
one-active-plan invariant when tracked and lightweight-local plans coexisted.
Updated `docs/specs/state-model.md` and `docs/specs/state-transitions.md` so
the selection and transition contracts now agree on those cases. The final
follow-up round, `review-004-delta`, then passed clean with `correctness` and
`docs_consistency`. Focused validation used
`rg -n "current-plan.json|one active|lightweight local plan|idle.*plan" docs/specs/state-model.md docs/specs/state-transitions.md`,
`go test ./internal/plan ./internal/status ./internal/lifecycle -count=1`,
and `harness plan lint docs/plans/active/2026-03-30-lightweight-workflow-for-low-risk-changes.md`.

### Step 2: Implement lightweight template and lifecycle behavior

- Done: [x]

#### Objective

Teach the CLI to author and manage lightweight plans and local archives without
breaking the existing standard-plan workflow.

#### Details

This step should cover the concrete command/runtime behavior implied by the new
contract:
- `harness plan template` can seed a lightweight plan mode such as
  `--lightweight`
- plan loading and linting understand optional `workflow_profile`
- the current-plan/runtime machinery can point at a local lightweight plan
- archive/closeout behavior keeps lightweight history in `.local/`
- status or related command output can surface breadcrumb reminders for
  lightweight changes at the right moment

Keep the implementation additive so standard tracked plans continue to work
unchanged unless they explicitly opt into the new profile field, and omitted
`workflow_profile` values continue to resolve as `standard`.

#### Expected Files

- `internal/plan/template.go`
- `internal/plan/template_test.go`
- `internal/plan/document.go`
- `internal/plan/document_test.go`
- `internal/plan/lint.go`
- `internal/plan/lint_test.go`
- `internal/plan/current.go`
- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `cmd/harness/main.go`

#### Validation

- The lightweight template mode produces a usable low-risk plan artifact.
- Standard tracked-plan behavior remains intact for existing tests and ordinary
  command flows.
- Status/archive behavior shows the intended lightweight closeout guidance.

#### Execution Notes

Added `internal/plan/profile.go` plus profile-aware helpers across template,
lint, runtime, current-plan detection, status, and lifecycle code so
lightweight plans can live under `.local/harness/plans/<plan-stem>/active/`
and archive into the matching local `archived/` directory without changing the
default standard tracked-plan behavior when `workflow_profile` is omitted.
`harness plan template` now supports `--lightweight`, seeds the optional
`workflow_profile: lightweight` field, and emits a shorter low-risk default
shape. Runtime behavior now treats a sole active lightweight local plan as
recoverable even without a current-plan pointer, validates lightweight paths
and profile values during lint, and surfaces repo-visible breadcrumb reminders
from lifecycle/status output during lightweight publish handoff. Focused
validation used `go test ./internal/plan ./internal/status ./internal/lifecycle ./internal/cli -count=1`
and later `go test ./... -count=1` after the end-to-end coverage landed.

#### Review Notes

`review-005-delta` passed clean across the `correctness` and `tests`
dimensions. Review specifically checked workflow-profile parsing,
current-plan resolution for tracked versus lightweight-local plans, archive
path selection, default-standard compatibility when `workflow_profile` is
omitted, and whether focused tests covered template, lint, runtime, lifecycle,
and status behavior. No findings were raised. Validation at closeout used
`go test ./internal/plan ./internal/status ./internal/lifecycle ./internal/cli -count=1`,
`go test ./tests/e2e -count=1`, and `go test ./... -count=1`.

### Step 3: Cover the workflow and teach agents to use it

- Done: [x]

#### Objective

Add end-to-end coverage and agent guidance so the lightweight path is both
tested and usable by future controllers.

#### Details

The repository should prove one realistic lightweight scenario, likely a
docs-only or README-scale change, from plan creation through local archive or
equivalent closeout. The agent-facing guidance should explain:
- how to decide between `standard` and `lightweight`
- that lightweight still requires human steering
- where the plan/archive artifacts live
- how to leave the repo-visible breadcrumb in the PR body or another approved
  review surface

If `harness status` supplies the breadcrumb reminder, make sure the docs and
tests both reinforce that behavior.

#### Expected Files

- `tests/e2e/`
- `.agents/skills/harness-plan/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/closeout-and-archive.md`
- `AGENTS.md`
- `README.md`

#### Validation

- At least one E2E or similarly high-signal workflow test covers the new
  lightweight path.
- The plan/execution docs tell a future agent how to choose and document the
  lightweight path without hidden chat context.

#### Execution Notes

Added `tests/e2e/lightweight_workflow_test.go` to exercise a realistic
lightweight local-plan flow end to end: template creation with
`--lightweight`, pre-execute status resolution, step review, finalize review,
local archive handoff, breadcrumb reminders during publish, and eventual
`execution/finalize/await_merge` after publish/CI/sync evidence. Updated
`tests/e2e/coverage_test.go` so the canonical transition catalog matches the
tracked state-transition matrix for the new lightweight idle-to-plan contract
without corrupting markdown cells that merely end with inline code, and
registered `TestLightweightWorkflowWithBuiltBinary` in the scenario coverage
catalog so the catalog now proves the lightweight-specific workflow remains
represented. Updated `.agents/skills/harness-plan/SKILL.md`,
`.agents/skills/harness-execute/SKILL.md`, and
`.agents/skills/harness-execute/references/closeout-and-archive.md` so future
agents know when `--lightweight` is allowed, that local lightweight plans
still need lint and human steering, and that archive-time breadcrumb guidance
remains mandatory. Validation used `go test ./tests/e2e -count=1` and
`go test ./... -count=1`.

#### Review Notes

`review-006-delta` checked the `tests` and `docs_consistency` dimensions.
`docs_consistency` passed clean, but `tests` raised one important finding:
`TestLightweightWorkflowWithBuiltBinary` existed without being registered in
`currentScenarioCoverage`, so the catalog would not have noticed if the new
lightweight scenario disappeared. Added the missing scenario entry and reran
`go test ./tests/e2e -count=1`. The bounded follow-up round,
`review-007-delta`, then rechecked the `tests` slot and passed clean with no
remaining findings.

## Validation Strategy

- Lint the tracked implementation plan with `harness plan lint`.
- Run focused unit tests for plan/template, lifecycle, and status behavior.
- Run at least one end-to-end lightweight workflow test plus the relevant
  existing workflow coverage needed to prove no regression in standard plans.
- Reread the updated docs and skills together to confirm the lightweight path
  remains explicit, bounded, and consistent.

## Risks

- Risk: A vague lightweight contract could become a loophole that lets
  substantive work bypass tracked-plan discipline.
  - Mitigation: Define eligibility and escalation rules explicitly, and make
    lightweight an opt-in profile instead of a fuzzy heuristic.
- Risk: Local-only plans and archives could become too invisible for reviewers
  or future agents.
  - Mitigation: Require a repo-visible breadcrumb and surface that requirement
    in status guidance and agent docs.
- Risk: Supporting both tracked and local plan locations could complicate
  current-plan resolution and archive behavior.
  - Mitigation: Add focused runtime and E2E coverage around plan selection,
    status, and archive transitions for both profiles.

## Validation Summary

Validated the tracked contract with `harness plan lint
docs/plans/active/2026-03-30-lightweight-workflow-for-low-risk-changes.md`,
focused package coverage with
`go test ./internal/plan ./internal/status ./internal/lifecycle ./internal/cli -count=1`,
targeted workflow coverage with `go test ./tests/e2e -count=1`, and the full
repository suite with `go test ./... -count=1`. The final focused rerun after
the Step 3 repair revalidated the lightweight scenario coverage catalog. After
publish handoff revealed stale sync against `origin/main`, reopened in
`finalize-fix`, merged `origin/main`, and reran `go test ./... -count=1` for
revision 2 before the second finalize review.

## Review Summary

Step closeout review passed for Step 2 in `review-005-delta`. Step 3 first ran
through `review-006-delta`, which found one important gap: the new lightweight
E2E was missing from `currentScenarioCoverage`. After registering that
scenario and rerunning the E2E suite, `review-007-delta` passed clean. The
full archive-candidate review, `review-008-full`, then passed clean across the
`correctness` and `risk_scan` dimensions with no remaining findings. After
reopen to revision 2 for stale sync against `origin/main`, merged the updated
mainline, reran full validation, and `review-009-full` again passed clean
across `correctness` and `risk_scan`.

## Archive Summary

- Archived At: 2026-03-31T00:55:41+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/79
- Ready: The candidate has clean step reviews, a clean finalize review, all
  acceptance criteria checked, and validation evidence recorded in the plan.
- Merge Handoff: Revision 2 restored sync freshness by integrating
  `origin/main`; after re-archive, update publish/CI/sync evidence for the
  current revision and wait for merge approval once CI is green and sync is
  fresh.

## Outcome Summary

### Delivered

Delivered an optional `workflow_profile` contract with default `standard`
behavior, lightweight template generation via `harness plan template
--lightweight`, local lightweight plan/archive path handling, breadcrumb-aware
status and lifecycle guidance, updated agent instructions, and end-to-end
coverage for the lightweight workflow.

### Not Delivered

Did not add direct CLI support for writing or validating lightweight
breadcrumbs, and did not add retrospective listing/reporting for archived
lightweight local plans.

### Follow-Up Issues

- #78: Add explicit CLI support for lightweight workflow breadcrumbs
- #77: Improve discoverability of archived lightweight local plans
