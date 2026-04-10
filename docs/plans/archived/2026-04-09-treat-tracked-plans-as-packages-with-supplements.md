---
template_version: 0.2.0
created_at: "2026-04-09T23:13:09+08:00"
source_type: direct_request
source_refs: []
---

# Treat tracked plans as packages with supplements

## Goal

Upgrade the tracked plan contract from a single markdown file to a plan
package whose main entrypoint remains `docs/plans/.../<stem>.md`, but whose
approved execution input can also include companion material under a sibling
`supplements/<stem>/` directory. The package should preserve discovery-time
detail that does not fit comfortably in the markdown plan itself, such as spec
drafts, structured design notes, formulas, or modeling details.

The package must still honor the current harness approval boundary. Human
approval covers the whole plan package, not only the markdown file. After that
approval, agents may keep execution-facing notes, closeout, and absorption
status current across the package, but must not silently change approved
intent, scope, acceptance criteria, or key design constraints without reusing
the existing plan-update or reopen approval path.

## Scope

### In Scope

- Define the new tracked-plan package contract in which active plans live at
  `docs/plans/active/<stem>.md` with companion material under
  `docs/plans/active/supplements/<stem>/...`, and archived standard plans live
  at `docs/plans/archived/<stem>.md` with archived companions under
  `docs/plans/archived/supplements/<stem>/...`.
- Update plan parsing, linting, and lifecycle rules so `archive` and `reopen`
  move the markdown plan and its matching supplements directory together as one
  package while still treating the markdown file as the primary plan path.
- Clarify the approval and execution contract for supplements so they share the
  same governance rules as the plan markdown: they are approved execution input
  during active work, and archived cold-storage context after archive.
- Add plan-template or plan-schema guidance for recording supplement
  absorption, such as which drafts were promoted into formal specs, code, or
  other durable repository locations before archive.
- Update status, timeline, review, and related command/UI read models only as
  needed so plan-package artifacts remain intelligible without changing the
  primary user entrypoint away from the markdown plan.
- Refresh the bootstrap guidance and repo docs so future agents know that plan
  packages, not chat history, are the durable carrier for discovery detail that
  must survive context compaction.

### Out of Scope

- Inventing a fine-grained taxonomy inside `supplements/` beyond what is needed
  to prove the package contract works.
- Turning archived supplements into the primary reading surface after archive;
  the archived markdown plan should remain the default entrypoint.
- Adding compatibility shims that preserve a single-file-only mental model when
  the new package semantics would be clearer.
- Migrating or backfilling every historical archived plan into the new package
  structure unless a targeted fixture update is required for tests.
- Reworking unrelated harness workflow semantics such as review orchestration,
  evidence capture, or merge gating beyond the package-path changes required by
  this slice.

## Acceptance Criteria

- [x] The normative plan contract clearly defines a tracked plan package as
      `<stem>.md` plus an optional-but-governed `supplements/<stem>/` tree for
      both active and archived standard-plan locations.
- [x] `harness archive` and `harness reopen` treat the markdown plan and its
      matching supplements tree as one package, moving both directions
      symmetrically while keeping the current-plan pointer centered on the
      markdown path.
- [x] Plan linting or equivalent package validation rejects mismatched
      supplement ownership or illegal active/archived placement rules, and
      continues to validate the markdown plan itself cleanly.
- [x] The documented approval boundary states that supplements are part of the
      approved plan package during active execution and follow the same
      change-governance rules as the plan markdown.
- [x] The plan/archive documentation and template guidance explain how to
      record supplement absorption so archived plans can summarize what was
      promoted into formal specs, code, or other durable repository artifacts.
- [x] Focused automated tests cover package-aware lifecycle behavior, including
      archive and reopen movement with and without supplements, plus any read
      models or command outputs that surface package artifacts.

## Deferred Items

- Automatic garbage collection or pruning rules for archived supplements beyond
  the first package-preserving archive contract.
- UI affordances that deeply browse supplement contents unless the basic
  package support forces a minimal read-model addition.
- Any future packaging of lightweight archived snapshots under
  `.local/harness/plans/archived/` unless this standard-plan slice proves a
  shared abstraction is obviously warranted.

## Work Breakdown

### Step 1: Define the plan-package contract and validation rules

- Done: [x]

#### Objective

Make the repository contract explicitly treat tracked plans as markdown-led
packages with governed supplements directories.

#### Details

Update the normative spec, template guidance, and plan-validation layer
together. The main questions are path shape, ownership, and governance: the
markdown file remains the canonical plan path, while `supplements/<stem>/`
belongs to that plan package and shares the same approval boundary. Validation
should catch supplements that do not match the current plan stem or that live
under the wrong active/archived root. Template or spec guidance should also
define where the plan records supplement absorption status so archive-time
readers can tell what was promoted into durable repo artifacts.

#### Expected Files

- `docs/specs/plan-schema.md`
- `docs/specs/index.md` if spec navigation needs an updated description
- `internal/plan/lint.go`
- `internal/plan/lint_test.go`
- `internal/plan/document.go` or related plan helpers if package metadata needs
  shared parsing support
- `internal/plan/template.go`
- `internal/plan/template_test.go`

#### Validation

- The plan-schema prose describes the package shape, approval semantics, and
  archive-time reading model without relying on discovery chat.
- `harness plan lint` accepts valid package-aware plans and rejects invalid
  supplement placement or ownership.
- Template guidance keeps the markdown concise while telling future agents how
  to track supplement absorption.

#### Execution Notes

Defined the markdown-led plan-package contract in `docs/specs/plan-schema.md`
so tracked plans may own matching `supplements/<plan-stem>/` directories under
the same active or archived root. Added package path helpers and lint rules in
`internal/plan/` to reject plan markdown stored under `supplements/`, accept
matching supplement directories, and require present supplement paths to be
directories. Revision 2 tightened the contract so archive-time correctness must
not depend on supplements remaining available verbatim, and lightweight plans
now explicitly avoid supplements by default even though the archive/reopen
mechanics still support them. Focused validation: `go test ./internal/plan`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The spec, helper, and lint changes are tightly coupled
to the runtime package movement in Step 2, so isolated step-closeout review
would be less meaningful than the final full-candidate review.

### Step 2: Make lifecycle and runtime state package-aware

- Done: [x]

#### Objective

Ensure lifecycle commands and plan-oriented read models move and report plan
packages coherently while preserving the markdown path as the primary runtime
handle.

#### Details

`archive` and `reopen` should move both the markdown file and its matching
`supplements/<stem>/` directory as one package. The current-plan pointer and
most command artifacts should still point at the markdown path so existing
mental models stay legible, but any package-aware artifact fields or helper
behavior needed for status, timeline, review, or UI inspection should be added
deliberately and consistently. This step should also decide whether package
helpers live in `internal/plan/` or a neighboring package rather than
re-implementing directory math in each lifecycle callsite.

#### Expected Files

- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `internal/plan/current.go`
- `internal/plan/runtime.go`
- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/timeline/service.go`
- `internal/timeline/service_test.go`
- `internal/reviewui/service.go`
- `internal/reviewui/service_test.go`
- `internal/cli/timeline_events.go` if surfaced artifact details need updates

#### Validation

- Archive and reopen tests prove markdown and supplements move together in both
  directions when the supplements tree exists.
- Plans without supplements still behave correctly and do not require empty
  placeholder directories.
- Status/timeline/review surfaces remain readable and consistent even though
  the underlying contract now recognizes a plan package.

#### Execution Notes

Updated lifecycle archive/reopen behavior so matching `supplements/<plan-stem>/`
directories move with the markdown plan package and roll back correctly on
post-mutation failure. Surfaced package companion paths through lifecycle and
status artifacts when a supplements directory exists, then covered the behavior
with focused lifecycle/status tests plus broader read-model regression coverage.
Validation: `go test ./internal/lifecycle ./internal/status ./internal/timeline
./internal/reviewui`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This runtime work is best reviewed together with the
spec/schema/doc updates in the final full review so the package contract can be
checked end to end.

### Step 3: Refresh workflow guidance, bootstrap assets, and repo docs

- Done: [x]

#### Objective

Teach future agents and humans that plan packages, not chat history, are the
durable home for discovery details that must survive compression.

#### Details

Update the plan workflow skill, bootstrap assets, and repository docs so they
describe the package contract consistently. The guidance should make three
things obvious: supplements are part of approval during active work, archived
markdown remains the primary reading surface after archive, and supplements are
expected to be absorbed into formal specs, code, or other durable docs rather
than left as forever-primary design material. Keep the docs aligned with the
repo's fast-development bias by choosing the clean end-state directly rather
than documenting compatibility stories.

#### Expected Files

- `README.md`
- `assets/bootstrap/skills/harness-plan/SKILL.md`
- `assets/bootstrap/agents-managed-block.md` if the tracked-plan description
  needs wording updates
- `.agents/skills/harness-plan/SKILL.md` via `scripts/sync-bootstrap-assets`
- root `AGENTS.md` managed block via `scripts/sync-bootstrap-assets`

#### Validation

- Workflow guidance tells a cold reader that discovery detail should be folded
  into the tracked plan package instead of hidden in chat memory.
- Bootstrap sync is clean after any asset edits.
- Repository docs and skill guidance use the same package vocabulary as the
  normative spec and lifecycle behavior.

#### Execution Notes

Refreshed `README.md`, `docs/specs/index.md`, `docs/specs/cli-contract.md`,
`assets/bootstrap/skills/harness-plan/SKILL.md`, and the managed block source
to describe supplements as part of the approved plan package and archive-time
cold backup. Synced dogfood outputs with `scripts/sync-bootstrap-assets` and
refreshed generated schemas with `scripts/sync-contract-artifacts`. Validation:
`scripts/sync-bootstrap-assets --check`,
`scripts/sync-contract-artifacts --check`, and
`harness plan lint docs/plans/active/2026-04-09-treat-tracked-plans-as-packages-with-supplements.md`.
Revision 2 also updated the authoring guidance to say that repository-facing
normative content must be absorbed out of supplements before archive, and that
lightweight plans should only use supplements exceptionally.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This documentation/bootstrap closeout is coupled to the
same contract slice and will be covered by the final full review.

## Validation Strategy

- Run `harness plan lint` on the tracked plan while drafting, then keep using
  it as a guardrail when template or package-validation logic changes.
- Run focused Go tests for `internal/plan`, `internal/lifecycle`,
  `internal/status`, `internal/timeline`, and `internal/reviewui` as the
  package contract lands.
- Re-run any contract- or bootstrap-sync checks required by the touched files,
  including `scripts/sync-bootstrap-assets --check` and any spec/schema checks
  the implementation updates require.
- Add or update lifecycle regression tests that explicitly cover the presence
  and absence of supplements so the new contract does not regress into
  single-file assumptions.

## Risks

- Risk: Package semantics could become half-adopted, with lifecycle behavior
  moving supplements but docs or lint rules still treating plans as markdown
  only.
  - Mitigation: land spec, validation, lifecycle behavior, and workflow
    guidance in the same slice and test both package-aware and markdown-only
    paths.
- Risk: Agents may interpret supplements as free-form scratch space and drift
  from the approved intent without a clear guardrail.
  - Mitigation: write the governance rule explicitly that supplements share the
    same approval boundary as the plan markdown and only execution-facing
    updates may be made autonomously.
- Risk: Read models may surface too much package detail and make archived plans
  noisier than they need to be.
  - Mitigation: keep the markdown file as the default entrypoint and expose
    package detail only where it materially helps archive, reopen, or audit
    clarity.

## Validation Summary

- `go test ./internal/plan ./internal/lifecycle ./internal/status` passed
  after revision 2 fixed the review-reported gaps around blocking supplements
  parent paths and lightweight no-supplements regression assertions.
- `scripts/sync-bootstrap-assets --check` and
  `scripts/sync-contract-artifacts --check` passed after refreshing the
  bootstrap-managed skill pack and generated contract artifacts for the
  tightened supplements guidance.
- `harness plan lint docs/plans/archived/2026-04-09-treat-tracked-plans-as-packages-with-supplements.md`
  passed after archive, confirming the archived tracked plan remains valid from
  the plan validator's perspective.

## Review Summary

- Finalize full review `review-007-full` requested changes on revision 2 for
  two blocking gaps: missing lint validation for blocking `supplements` parent
  paths, and missing negative assertions that lightweight archive/status keep
  supplements absent by default when no supplements directory exists.
- Follow-up repairs added the parent-path lint guard plus the lightweight
  archive/status regression assertions, then finalize full review
  `review-008-full` passed on 2026-04-10 with zero blocking and zero
  non-blocking findings.

## Archive Summary

- Archived At: 2026-04-10T08:46:46+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/127
- Ready: Revision 2 keeps supplements as approved execution input during active
  work but makes archive-time correctness independent from archived supplements
  remaining present, requires normative content to be absorbed into formal
  tracked locations before archive, keeps lightweight supplements exceptional
  and local-only, and passed finalize full review `review-008-full`.
- Merge Handoff: Run `harness archive`, commit the tracked archive move plus
  revision 2 closeout summaries, push the branch, refresh PR #127, record
  updated publish/CI/sync evidence, and wait for explicit human merge approval
  once `harness status` reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Upgraded tracked plans from single markdown files to markdown-led packages
  that may own matching `supplements/<plan-stem>/` trees in both active and
  archived locations, including the lightweight archived snapshot path.
- Taught plan lint, lifecycle archive/reopen behavior, and status/lifecycle
  artifacts to understand those package companions while keeping the markdown
  file as the primary runtime handle.
- Updated the normative specs, checked-in schemas, plan template guidance, and
  bootstrap-managed workflow docs so future agents treat supplements as part
  of the approved plan package and record archive-time absorption clearly.
- Tightened the supplements contract so anything the repository must still
  depend on after archive gets absorbed into formal tracked locations, while
  lightweight plans avoid supplements by default and keep any archived
  companion snapshot under `.local/` only.
- Added regression coverage for the revision 2 guardrails by asserting that
  blocking `supplements` parent paths fail lint and that lightweight
  archive/status flows keep supplements artifacts absent when no supplements
  directory exists.

### Not Delivered

- No new deep UI affordances for browsing supplement contents were added in
  this slice.
- No taxonomy inside `supplements/` beyond the shared root and per-plan stem
  ownership contract was introduced.
- No automatic pruning or deletion policy for archived supplements was added in
  this slice; archived supplements remain cold backup context when they exist.

### Follow-Up Issues

- No new follow-up issue was created in this slice. The deferred items remain
  future workflow and UX work around supplement browsing, pruning, or richer
  packaging policy if the repository later decides those tradeoffs are worth
  formalizing.
