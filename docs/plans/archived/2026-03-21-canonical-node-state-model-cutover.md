---
template_version: 0.2.0
created_at: "2026-03-21T00:00:00+08:00"
source_type: direct_request
source_refs: []
---

# Cut over to the v0.2 canonical-node state model

## Goal

Replace the current layered v0.1 lifecycle, step-state, handoff-state, and
worktree-state model with the v0.2 canonical `current_node` model across the
tracked plan contract, CLI commands, `.local` runtime artifacts, and status
rendering.

This cutover should leave the repository with formal v0.2 specs under
`docs/specs/`, including a dedicated transition spec that enumerates every
allowed state transition, plus a dogfood-ready workflow that this repository
can start using before the entire branch is finished once the core CLI path is
coherent enough to switch.

## Scope

### In Scope

- Promote the state-model v0.2 proposal into the formal spec set under
  `docs/specs/` and stop relying on the proposal file as the primary source of
  truth.
- Add a dedicated state-transition spec that enumerates every allowed
  `current_node` transition, including the driver, preconditions, owned
  artifacts, and resulting node.
- Replace the tracked plan schema so top-level execution state no longer lives
  in frontmatter and step progress uses durable `Done` markers instead of
  `Status:` lines.
- Introduce the agreed v0.2 command surface:
  `status`, `execute start`, `review start|submit|aggregate`,
  `evidence submit`, `archive`, `reopen`, `land`, and `land complete`.
- Make `ci`, `publish`, and `sync` evidence command-owned append-only
  trajectory artifacts with explicit `not_applied` support.
- Make `state.json` a CLI-owned thin cache that stores `current_node` and
  latest artifact pointers rather than duplicating full runtime facts.
- Rewrite `harness status` so its JSON contract is pure v0.2 and derives
  summary plus next actions from `current_node`.
- Keep `archive` as the finalize freeze gate after clean finalize review, while
  moving PR, CI, freshness, and conflict evidence into
  `execution/finalize/publish`.
- Implement explicit reopen modes so the CLI can distinguish
  `reopen --mode finalize-fix` from `reopen --mode new-step` without relying
  on hidden agent judgment.
- Update operator-facing docs and repo-local skills so `README.md`,
  `AGENTS.md`, and the harness skills all describe the same v0.2 workflow.
- Dogfood the cutover in this repository as soon as the core CLI and status
  path are usable, even if some cleanup work remains afterward.

### Out of Scope

- Preserving a v0.1 compatibility layer in CLI output, plan schema, or local
  state.
- Supporting multiple simultaneous active plans in one repository.
- Adding step-level completion commands beyond durable plan edits.
- Teaching harness to perform `git`, push, PR creation, merge, comment, or
  issue-closing actions on the agent's behalf.
- Supporting non-PR land flows in this cutover; v0.2 land entry will require a
  PR URL.

## Acceptance Criteria

- [x] Formal v0.2 state-model docs live under `docs/specs/`, and a dedicated
      transition spec enumerates every allowed `current_node` transition with
      no need to reconstruct behavior from discovery chat.
- [x] The tracked plan contract, template, parser, and lint rules enforce the
      v0.2 shape: no top-level execution frontmatter, durable `Done` markers,
      and reopen placeholders that preserve archive audit history instead of
      blanking it out.
- [x] The CLI exposes the agreed v0.2 command surface, including
      `execute start`, `evidence submit`, `reopen --mode <...>`,
      `land --pr <url> [--commit <sha>]`, and `land complete`, with help text
      that includes the evidence JSON schema and `not_applied` examples.
- [x] Runtime artifacts are command-owned where trajectory matters, with
      append-only evidence history for `ci`, `publish`, and `sync`, while
      `state.json` remains a thin cache containing only `current_node` and
      latest artifact pointers.
- [x] `harness status` emits pure v0.2 JSON centered on `current_node`,
      selected facts, summary, and next actions, with exhaustive automated
      coverage for every node and every transition enumerated in the dedicated
      transition spec.
- [x] `README.md`, `AGENTS.md`, and repo-local skills consistently describe the
      v0.2 workflow, and dogfooding in this repository proves that an
      in-progress v0.1-style workspace can cut over once the new CLI path is
      ready enough to use.

## Deferred Items

- Non-PR merge/land support remains deferred; the v0.2 cutover only supports
  `harness land --pr <url> [--commit <sha>]`.
- Automatic migration tooling for arbitrary historical v0.1 `.local` state is
  deferred unless the planned dogfood exposes a concrete need beyond the
  repository's supported cutover path.

## Work Breakdown

### Step 1: Promote v0.2 into formal specs and enumerate every transition

- Done: [x]

#### Objective

Turn the v0.2 proposal into formal repository specs and add a dedicated
transition spec that explicitly enumerates every allowed `current_node`
transition.

#### Details

Rewrite `docs/specs/state-model.md` so it is the normative v0.2 state-model
contract rather than a v0.1 descriptive map. Add a separate
`docs/specs/state-transitions.md` spec that lists every state transition,
including: source node, destination node, trigger, required command or
artifact, validation/preconditions, and any transition-specific notes such as
reopen mode behavior. Update the spec index so a future agent can find the
formal docs first, and remove the adopted proposal file so the formal specs
are the only source of truth.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/state-transitions.md`
- `docs/specs/index.md`

#### Validation

- A cold reader can answer "what are all the nodes?" and "what are all the
  allowed transitions?" from the formal spec files alone.
- The transition spec explicitly covers execute start, step review loops,
  finalize review/archive/publish/await-merge progression, both reopen modes,
  and land entry plus completion.

#### Execution Notes

Promoted the canonical-node design into formal tracked specs by rewriting
`docs/specs/state-model.md` as the normative v0.2 state-model contract and
adding a dedicated `docs/specs/state-transitions.md` file that enumerates
every allowed `current_node` transition, including reopen and land flows.
Updated the spec index to point readers at the new formal docs first and
removed the superseded state-model proposal file so the formal specs are the
only source of truth. Tightened the step loop so `execution/step-<n>/review`
means review is actively in flight, aggregated step outcomes always return to
`execution/step-<n>/implement`, and step-level repair no longer has a separate
`fix` node. Added explicit guidance that commits support reviewability and
handoff but do not change `current_node` on their own.

#### Review Notes

No dedicated step review round was run for this docs-only slice. The new
formal specs were cross-checked against the approved discovery decisions, and
the broader branch review will still cover wording, omissions, and contract
consistency before archive.

### Step 2: Replace the tracked plan contract with durable completion markers

- Done: [x]

#### Objective

Cut the tracked plan schema over to the v0.2 contract so plans carry durable
scope, closeout notes, and step completion markers without top-level execution
state.

#### Details

Update the formal plan-schema spec, packaged plan template, plan parser, and
plan lint rules together. Replace step `Status:` lines with durable `Done`
markers, remove the top-level execution-state frontmatter fields, and preserve
archive/reopen audit history through explicit update-required placeholders
instead of blank resets. Keep the file-move-based active versus archived split,
but stop treating tracked plan frontmatter as the runtime state machine.

#### Expected Files

- `docs/specs/plan-schema.md`
- `assets/templates/plan-template.md`
- `internal/plan/template.go`
- `internal/plan/document.go`
- `internal/plan/document_test.go`
- `internal/plan/lint.go`
- `internal/plan/lint_test.go`

#### Validation

- `harness plan template` renders the new plan shape.
- `harness plan lint` accepts valid v0.2 plans and rejects legacy execution
  state fields, missing `Done` markers, and invalid reopen placeholder usage.
- Plan parsing still identifies the first unfinished step correctly from the
  new durable completion markers.

#### Execution Notes

Step 2 now hard-cuts the tracked plan contract instead of treating v0.2 as a
mixed migration. The packaged template renders only the durable frontmatter
fields (`template_version`, `created_at`, `source_type`, `source_refs`) and
every step uses `- Done: [ ]` / `- Done: [x]`. Lint now rejects legacy
runtime frontmatter (`status`, `lifecycle`, `revision`, `updated_at`) and
legacy step `Status:` lines, while `LoadFile` still parses old historical
documents when it needs to read them. The lifecycle/status/review services now
derive active-vs-archived from plan path, treat explicit runtime milestones as
the source of execution state, and keep revision in command-owned runtime
state instead of tracked frontmatter. Reopen also preserves archive-time audit
history by prepending `UPDATE_REQUIRED_AFTER_REOPEN` markers instead of
blanking summaries back to fresh placeholders. Archive no longer treats CI or
sync freshness as pre-archive gates; those facts remain for later publish
readiness.

#### Review Notes

Review rounds `review-001-delta` and `review-002-delta` surfaced three real
migration hazards: mixed `Done`/legacy `Status` plans could resolve the wrong
current step in either direction, the temporary lint change got ahead of
downstream lifecycle consumers, and coverage was missing those edge cases.
Addressed all three, then reran delta review as `review-003-delta`, which
aggregated clean with a pass decision. A later focused round,
`review-004-delta`, found rollback and reopen-mode gaps around `execute
start`; those were fixed before `review-005-delta` reran clean with a pass
decision. Follow-up round `review-006-delta` caught two lifecycle regressions
(`review start` could bypass explicit execution-start state and archive had
slipped from finalize-review gating) plus two missing tests around the
publish-independent archive gate and reopen marker clearing. After those fixes,
`review-007-delta` narrowed the remaining gap to one test still carrying
publish evidence, and `review-008-delta` reran clean once the archive success
case explicitly proved publish evidence is absent. This step is now ready for
dogfood through the active tracked plan's own v0.2 shape.

### Step 3: Introduce v0.2 milestone commands and trajectory artifacts

- Done: [x]

#### Objective

Implement the new command-owned milestone and evidence surface that the v0.2
 runtime model depends on.

#### Details

Add `harness execute start`, `harness evidence submit`, `harness land`, and
`harness land complete`, and reshape `harness reopen` to require an explicit
mode such as `finalize-fix` or `new-step`. Keep review artifacts command-owned
and add append-only evidence artifacts for `ci`, `publish`, and `sync`, with
explicit `not_applied` handling instead of missing-value guesses. CLI help for
`evidence submit` should include schema details and examples for each kind. The
runtime cache should stay thin: latest artifact pointers plus the last resolved
`current_node`, not a duplicated fact snapshot.

#### Expected Files

- `docs/specs/cli-contract.md`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/runstate/state.go`
- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `internal/review/service.go`
- `internal/review/service_test.go`
- `internal/evidence/service.go`
- `internal/evidence/service_test.go`

#### Validation

- Command tests cover execute-start milestones, evidence submission for all
  supported kinds, explicit `not_applied`, reopen mode parsing, land entry with
  `--pr` and optional `--commit`, and land completion back to idle.
- Append-only evidence artifacts preserve timestamps and sufficient trajectory
  detail to reconstruct what the agent observed and when.

#### Execution Notes

Implemented the first real v0.2 command-surface slice. Added
`internal/evidence/service.go` with append-only `ci`, `publish`, and `sync`
artifacts plus latest evidence pointers in `state.json`; kept the old
`LatestCI` / `LatestPublish` / `Sync` cache fields only as a transitional shim
so `status` can keep working until Step 4 rewrites it around `current_node`.
Replaced the user-facing land command surface with `harness land --pr <url>
[--commit <sha>]` and `harness land complete`, including readiness checks that
read the latest evidence artifacts before entering cleanup. Reinstalled the dev
binary and dogfooded the new root/help surface (`harness --help`,
`harness evidence submit --help`, `harness land --help`) so the installed
command now exposes the intended Step 3 entry points. Also updated the repo's
most obvious operator docs (`README.md`, `AGENTS.md`, `harness-land`, and the
publish/CI/sync execute reference) so they no longer instruct agents to use the
removed `land record` flow. A temporary shell dogfood flow in a temp repo also
proved `evidence submit -> land -> land complete -> status` works end to end;
that dry run exposed one JSON-envelope polish issue (empty `land` state fields),
which was fixed before the final confirmation review.

#### Review Notes

`review-009-delta` reviewed the initial Step 3 slice across correctness, tests,
and docs consistency. It surfaced six real issues: four coverage gaps
(publish/sync success-path evidence coverage, a fake execute-start backfill
fixture, missing reopen assertions for land/current-node, and no negative
coverage for `land complete`) plus two correctness gaps (no rollback when
`land complete` fails after updating `state.json`, and no legacy cache fallback
for `land` readiness during the cutover). All six were fixed, then
`review-010-delta` reran the follow-up correctness/tests slice clean with a
pass decision. Step 3 is now reviewed and ready to hand off to the pure
`current_node` status rewrite in Step 4.

### Step 4: Resolve `current_node` and rewrite `harness status`

- Done: [x]

#### Objective

Replace the layered v0.1 status logic with a single current-node resolver and a
pure v0.2 status JSON contract.

#### Details

Teach status to resolve the canonical node from the tracked plan, execute
 milestone, review artifacts, evidence artifacts, archive/reopen milestones,
 and land milestones. Remove `plan_status`, `lifecycle`, `step_state`,
 `handoff_state`, and `worktree_state` from the JSON contract. Keep selected
 facts, summary text, and next actions, but make them derive from
 `current_node` plus the latest relevant artifacts. Exhaustive transition
 coverage matters here: the implementation and tests should line up directly
 with the dedicated transition spec rather than relying on ad hoc branching.

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/runstate/state.go`
- `internal/plan/current.go`
- `internal/plan/current_test.go`

#### Validation

- Automated coverage proves every formal node can be reached and that every
  transition listed in `docs/specs/state-transitions.md` resolves to the
  expected node.
- `harness status` returns pure v0.2 JSON with actionable next actions for
  `plan`, step implement/review, finalize review/fix/archive/publish/await
  merge, `land`, and `idle`.

#### Execution Notes

Rewrote `internal/status/service.go` around a single canonical
`current_node` resolver and cut the JSON payload over to `state.current_node`
plus selected `facts`, `summary`, `next_actions`, `artifacts`, `warnings`,
and `blockers`. The new resolver reads tracked plan progress, execution-start,
review artifacts, append-only evidence, reopen milestones, and land markers,
then refreshes `state.json` as a thin cache of the resolved node rather than
reconstructing v0.1 lifecycle fields. Added review-manifest target recovery in
`runstate` so failed step reviews pin status back to the reviewed step instead
of letting a prematurely-checked `Done` marker advance to the next step or
finalize. Tightened `plan.ExecutionStarted` so a cached `current_node=plan`
does not falsely flip the active plan into `executing`.

While dogfooding the new resolver against this repository's own local state,
`harness status` initially misinterpreted an aggregated `review_fix` round from
Step 3 as a clean review for Step 4. Fixed that by treating only structural
review triggers (`step_closeout` and `pre_archive`) as canonical state inputs;
non-structural review rounds can still exist in local artifacts, but they no
longer leak into node resolution or step summaries. Also extended reopen state
with `base_step_count` so `reopen --mode new-step` can distinguish "new step
still needs to be added" from "a new step was added and later completed"
without getting stuck forever in reopen-pending logic.

Replaced the old v0.1-shaped status tests with node-centric coverage for:
`plan`, step implement/review, failed-step pinning, finalize review/fix/archive,
archived publish/await-merge, `land`, `idle`, both reopen modes, and the
non-structural-review dogfood regression. Manual smoke confirmed the installed
binary now reports this repository as
`execution/step-4/implement` with only `current_step` in `facts`, instead of
reusing Step 3's stale `review_fix` metadata.

Follow-up review-driven tightening made unsafe review recovery fully
conservative: if step review metadata is missing or the recorded target does
not match a tracked step, status now pins the most likely reviewed step back
to `implement`, suppresses structural review facts, and skips refreshing the
`current_node` cache instead of treating guessed step recovery as canonical.
Added focused coverage for mismatched `step_closeout` targets and for the
legacy `latest_publish` / `latest_ci` / `sync` evidence fallback so archived
status still resolves correctly while old dogfood artifacts remain in play.

#### Review Notes

Delta review rounds `review-011-delta` through `review-016-delta` drove the
status rewrite to a clean slice. Early rounds caught reviewed-step recovery
bugs when manifests or triggers were missing, stale `review_fix` rounds were
being treated as structural state, `land` could overwrite merge confirmation,
and publish/sync dirty-path coverage was incomplete. The final follow-up round,
`review-015-delta`, tightened the remaining unsafe-fallback contract by
requiring guessed step recovery to stay in `implement`, suppress structural
review facts, and skip `current_node` cache refresh; it also required explicit
coverage for mismatched review targets and the legacy evidence fallback path.
After those fixes, `review-016-delta` reran clean with pass decisions from both
the correctness and tests reviewers.

### Step 5: Update README, AGENTS, and repo-local skills for v0.2

- Done: [x]

#### Objective

Bring the human-facing docs and repo-local skills into alignment with the new
v0.2 state model and command surface.

#### Details

Update `README.md`, `AGENTS.md`, and the relevant repo-local skills so a future
agent or human operator learns the v0.2 workflow from tracked docs rather than
obsolete v0.1 assumptions. This includes the new status model, command-owned
evidence flow, explicit reopen modes, land flow, dogfood expectations, and the
fact that all state transitions are now enumerated in the dedicated transition
spec. Touch every skill or durable doc that still instructs agents to reason in
terms of v0.1 lifecycle fields or old local-state ownership.

#### Expected Files

- `README.md`
- `AGENTS.md`
- `.agents/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-plan/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-land/SKILL.md`
- `.agents/skills/harness-reviewer/SKILL.md`

#### Validation

- A future agent following only the updated docs and skills will use the v0.2
  commands, spec files, and state vocabulary instead of the removed v0.1
  fields.
- Doc wording agrees with the formal spec set and the implemented CLI help.

#### Execution Notes

Updated the operator-facing docs and repo-local skills to speak in v0.2
canonical-node terms instead of the removed v0.1 layered status vocabulary.
`README.md` and `AGENTS.md` now describe the workflow around
`state.current_node`, the archive-to-publish handoff, and the two-stage land
flow. The repo-local skills were aligned the same way: `harness-execute`
now keys off node boundaries instead of lifecycle/handoff fields,
`harness-land` starts only from `execution/finalize/await_merge`,
`harness-plan` exits with the plan ready to resolve to `plan`, and
`harness-discovery` / `harness-reviewer` no longer describe their handoffs in
legacy lifecycle terms. The execute references for archive and
publish/CI/sync were updated to match the post-archive evidence flow, and
`docs/specs/cli-contract.md` now specifies the pure v0.2 `harness status`
envelope plus the archive/reopen contracts that go with it. Follow-up doc
repairs after review made every surfaced reopen path explicit about
`--mode finalize-fix|new-step` and clarified that a clean post-land resume may
legitimately start from `idle` with no current tracked plan to open.

#### Review Notes

`review-017-delta` reviewed the first doc-and-skill alignment pass across docs
consistency and agent UX. It found that several surfaced command references
still treated `reopen` as a bare command and that the resume instructions still
assumed a current tracked plan always existed. Follow-up edits made every
reopen path explicit about `--mode finalize-fix|new-step` and clarified that a
post-land `idle` resume may legitimately have no current plan to open.
`review-018-delta` then reran clean with pass decisions from both the
docs-consistency and agent-ux reviewers.

### Step 6: Dogfood the cutover in this repository and close migration gaps

- Done: [x]

#### Objective

Exercise the new v0.2 flow in this repository as soon as the CLI path is ready
enough to switch, then use what the dogfood reveals to finish the cutover
cleanly.

#### Details

Do not wait for every cleanup detail before trying the new flow. Once the
core command surface, runtime artifacts, and status resolution are coherent,
switch this repository's active work onto the v0.2 path and record what breaks
or feels awkward when real plans and `.local` artifacts still carry v0.1
history. Use the dogfood results to tighten the supported cutover path, close
migration gaps, and capture any deferred follow-up that genuinely deserves a
later issue rather than an incomplete cutover.

#### Expected Files

- `docs/plans/active/2026-03-21-canonical-node-state-model-cutover.md`
- `.local/harness/current-plan.json`
- `.local/harness/plans/<plan-stem>/`

#### Validation

- Real dogfood in this repository reaches the new command surface before the
  branch is done and demonstrates that a v0.1-style local state can be cut
  over or deliberately replaced in a documented way.
- `go test ./...` passes after the dogfood fixes land.
- Manual smoke checks cover `harness status`, `harness execute start`,
  `harness evidence submit`, `harness archive`, `harness reopen`, and the
  two-stage land flow.

#### Execution Notes

Dogfooded the cutover in two layers. First, this repository itself has now
been running the active plan, review rounds, and `harness status` on the pure
v0.2 path for multiple steps without any top-level tracked lifecycle fields or
`current-plan.json` pointer. That proved an in-flight worktree can cut over to
the canonical-node model mid-branch and continue executing from tracked plan
plus `.local` artifacts alone.

Second, ran destructive milestone smokes in temporary repositories so the live
tracked plan in this worktree would not be disrupted. The dogfood covered:

- `plan -> execution/step-1/implement` via `harness execute start`
- `execution/finalize/archive -> execution/finalize/publish` via
  `harness archive`
- `execution/finalize/publish -> execution/finalize/await_merge` via explicit
  publish, CI, and sync evidence submission
- `execution/finalize/await_merge -> land -> idle` via
  `harness land --pr <url> --commit <sha>` and `harness land complete`
- `execution/finalize/await_merge -> execution/finalize/fix` via
  `harness reopen --mode finalize-fix`
- `execution/finalize/await_merge -> execution/finalize/fix ->
  execution/step-3/implement` via `harness reopen --mode new-step` followed by
  adding a new unfinished step

An initial review follow-up pointed out that those smokes still had not proven
the legacy fallback path for upgraded local state. Added a dedicated temporary
repo scenario that archived a candidate, replaced the new evidence pointers
with legacy `latest_publish` / `latest_ci` / `sync` fields, then confirmed
`harness status` still resolved `execution/finalize/await_merge` and that
`harness land --pr ...` plus `harness land complete` still worked from that
legacy state. The only issue surfaced during the smokes was in the smoke
script itself: the first `new-step` run used `python` instead of `python3`
while appending the new step block. The harness commands themselves behaved as
expected, and the rerun with `python3` completed cleanly. The resulting
supported cutover path is now explicit: use the live worktree to prove
mid-flight status/review continuity, use isolated temporary repos to exercise
archive/reopen/land milestones end to end without disturbing the active
tracked plan, and include one legacy-state smoke when validating the
transition from pre-v0.2 local artifacts.

#### Review Notes

`review-019-delta` reviewed the first dogfood closeout pass and caught one real
gap: the notes overstated migration coverage because the temporary-repo smokes
had not yet exercised the legacy `latest_publish` / `latest_ci` / `sync`
fallback path that still exists for upgraded local state. Added a dedicated
legacy-state smoke to prove `status -> await_merge -> land -> idle` still
works from those older cache fields, then reran `review-020-delta`, which
passed clean across both correctness and agent-ux.

## Validation Strategy

- Drive the cutover spec-first, then keep package-level tests and command-level
  tests aligned with the enumerated transition matrix instead of adding ad hoc
  branches without spec coverage.
- Run targeted Go tests after each major slice, then finish with `go test ./...`
  plus direct CLI smoke checks for the new v0.2 workflow.
- Dogfood early enough that mixed old/new local state can still influence the
  implementation instead of being discovered only after the branch looks done.

## Risks

- Risk: Hidden v0.1 assumptions may survive in parser, lint, status, or
  repo-local skills and make the hard cutover feel complete before it really
  is.
  - Mitigation: Formalize the transition matrix first, then use exhaustive
    tests plus doc/skill updates to remove every known v0.1 dependency.
- Risk: Evidence and milestone artifacts could become another layered state
  machine if the cache starts duplicating facts or if missing evidence is
  treated as implicit success.
  - Mitigation: Keep `state.json` thin, make evidence append-only, and require
    explicit `not_applied` submissions instead of fallback guesses.

## Validation Summary

Validated the cutover with repeated package-level and full-suite runs of
`go test ./...` throughout the implementation, including final focused reruns
for the reopen guard, pointer, and CLI error-path follow-ups. After Go runtime
changes, the dev binary was reinstalled with `scripts/install-dev-harness
--force`, and the installed `harness` binary was exercised with `harness
status` plus command-help checks for the new v0.2 surface.

Dogfood covered two layers. This repository itself ran the active tracked plan,
review rounds, and `harness status` on the v0.2 path without tracked lifecycle
frontmatter. Separate temporary-repo smokes exercised `execute start`,
`archive`, append-only publish/CI/sync evidence, both reopen modes,
`harness land --pr ...`, and `harness land complete`, including a legacy-cache
fallback scenario for older `latest_publish` / `latest_ci` / `sync` state.

## Review Summary

The cutover was reviewed incrementally across the branch. Delta rounds
`review-001-delta` through `review-020-delta` closed plan-schema, lifecycle,
status, docs, and dogfood gaps step by step. Pre-archive full rounds
`review-021-full` through `review-029-full` then tightened the final archive
candidate around evidence gating after land entry, explicit reopen semantics,
reopen CLI branch coverage, the land-cleanup reopen guard, phase terminology,
and current-plan pointer coverage.

The final archive candidate passed clean structural review in
`review-029-full`. Its only non-blocking note was a stale `land record`
reference in the non-normative proposal
`docs/specs/proposals/testing-structure.md`, which was corrected immediately
before archive.

## Archive Summary

- Archived At: 2026-03-22T01:30:30+08:00
- Revision: 1
- PR: NONE
- Ready: The v0.2 cutover candidate satisfies the tracked acceptance criteria
  and is ready to be archived locally, then advanced through post-archive
  publish, CI, and sync evidence toward `execution/finalize/await_merge`.
- Merge Handoff: Commit and push the archived plan move, open or refresh the
  PR, then record publish, CI, and sync evidence until `harness status`
  resolves `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Formal v0.2 state-model and transition specs now live under `docs/specs/`,
  including exhaustive transition enumeration.
- Tracked plans now use durable `Done` markers, v0.2 reopen markers, and
  archive-time summaries instead of top-level runtime frontmatter.
- The CLI and runtime now use the agreed v0.2 command surface, append-only
  evidence artifacts, thin `state.json` caching, and canonical `current_node`
  resolution.
- README, AGENTS, and repo-local skills now describe the same v0.2 workflow
  and terminology.
- This repository successfully dogfooded the cutover, including temp-repo
  milestone smokes and legacy-cache fallback coverage.

### Not Delivered

- Non-PR land flows remain unsupported; `harness land` still requires
  `--pr <url>`.
- General migration tooling for arbitrary historical v0.1 `.local` state was
  not implemented in this cutover.

### Follow-Up Issues

- #19 Support non-PR land flows in the v0.2 state model.
- #20 Evaluate migration tooling for historical v0.1 local state.
