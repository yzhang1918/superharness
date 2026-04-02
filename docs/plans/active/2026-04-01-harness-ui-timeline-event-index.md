---
template_version: 0.2.0
created_at: "2026-04-01T21:14:39+08:00"
source_type: issue
source_refs:
    - '#93'
---

# Hook real timeline data into the harness UI with a command-owned event index

## Goal

Replace the `Timeline` page WIP placeholder with a real read-only view of the
current plan's harness trajectory. This slice should keep the existing
artifact-first product stance while avoiding a brittle "scan many directories
and guess" implementation.

The accepted direction is to add one thin command-owned event index per plan at
`.local/harness/plans/<plan-stem>/events.jsonl`. Relevant commands will append
timeline events that reference their existing artifacts rather than duplicating
full payloads. The UI will read a dedicated read-only timeline resource and
render ordered entries, transitions, and high-signal artifact links from that
index.

## Scope

### In Scope

- Define the new command-owned runtime artifact contract for
  `.local/harness/plans/<plan-stem>/events.jsonl`.
- Keep the existing CLI command names, flags, plan schema, and repo-local
  skills unchanged while extending runtime artifacts and read-only UI data.
- Append timeline events from the relevant existing commands:
  `execute start`, `archive`, `reopen`, `land`, `land complete`,
  `review start`, `review submit`, `review aggregate`, and
  `evidence submit`.
- Add the minimum read-only timeline API/resource needed by `harness ui`.
- Replace the `Timeline` page placeholder with real timeline rendering backed
  by the new event index.
- Add focused Go/unit coverage plus browser validation for event append, data
  loading, and timeline rendering.
- Sync any contract/schema artifacts required by the new runtime contract.

### Out of Scope

- Changing the `harness` command surface, subcommand names, or existing CLI
  flag contract.
- Changing harness workflow skills or requiring agents to adopt new command
  usage patterns.
- Backward-compatibility shims, historical backfill, or mixed old/new timeline
  readers for pre-index worktrees.
- UI-triggered mutations or direct action controls.
- Live data hookup for `Review`, `Diff`, or `Files`.
- Remote PR/CI integrations beyond the existing local evidence artifacts.

## Acceptance Criteria

- [x] A documented runtime artifact contract exists for
      `.local/harness/plans/<plan-stem>/events.jsonl`, and the checked-in
      contract artifacts/schema registry are updated accordingly.
- [x] The relevant lifecycle, review, and evidence commands append one ordered
      event each to the current plan's event index without changing their
      command-line usage.
- [x] Timeline events reference existing command-owned artifacts and include
      enough metadata to render time, kind, summary, revision, and state
      transition context without inventing a second canonical state model.
- [x] `harness ui` exposes a read-only timeline resource and the `Timeline`
      page renders real entries instead of the current WIP placeholder.
- [x] The timeline UI shows high-signal event details, including transitions
      and artifact references, while staying read-only and grounded in existing
      harness-owned files.
- [x] Focused tests cover event append behavior, timeline resource loading, and
      UI rendering/navigation for the live `Timeline` page.

## Deferred Items

- Rich document viewers or inline tabbed artifact inspection for review
  submissions, aggregates, or evidence record bodies beyond the first
  high-signal timeline presentation.
- Backfill or migration of older local worktrees that predate `events.jsonl`.
- Any reuse of the event index for `Review`, `Diff`, or `Files` beyond what is
  needed to land this timeline slice cleanly.

## Work Breakdown

### Step 1: Define the event-index contract and timeline resource boundary

- Done: [x]

#### Objective

Write the accepted direction back into tracked docs and contracts so a future
agent can implement or review the slice without relying on discovery chat.

#### Details

This step should lock the non-goals that were accepted during discovery:
command names stay unchanged, repo-local skills stay unchanged, and the new
timeline support must not become a second hidden state engine. The event index
is a command-owned runtime artifact that records ordered, append-only event
references. It should store only minimal event metadata plus references to
already-owned artifacts such as plans, review manifests/aggregates, or evidence
records.

This step should also define the read-only UI boundary: the frontend should not
scan `.local/harness` directly. Instead, Go should expose one thin timeline
resource/view model tailored for the current worktree and current plan.

#### Expected Files

- `docs/plans/active/2026-04-01-harness-ui-timeline-event-index.md`
- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- `docs/specs/contract.md`
- `internal/contracts/registry.go`
- new or updated contract type files under `internal/contracts/`

#### Validation

- The tracked docs make it clear that command usage stays the same while the
  runtime artifact surface grows.
- The contract registry and checked-in schema artifacts describe the new event
  index and any new timeline response shape.
- A future agent can explain why `events.jsonl` is an index and not a second
  canonical state source.

#### Execution Notes

Added a new Go-owned timeline contract module under `internal/contracts/` for
the append-only `events.jsonl` line shape plus the read-only `/api/timeline`
resource payload. Updated the registry entry list so contract sync now
generates `schema/artifacts/timeline-event.schema.json` and
`schema/ui-resources/timeline.schema.json`, then refreshed the checked-in
schema registry with `scripts/sync-contract-artifacts`. The prose specs now
record that `harness ui` exposes live `Status` and `Timeline` resources, that
timeline history stays grounded in command-owned runtime artifacts, and that
`.local/harness/plans/<plan-stem>/events.jsonl` is a CLI-owned runtime index
rather than a second canonical state source.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this contract-boundary step was developed as part of one
integrated timeline slice and will receive a full finalize review before
archive.

### Step 2: Append timeline events from command-owned runtime paths

- Done: [x]

#### Objective

Make the relevant commands emit one append-only timeline event each so the
current plan's trajectory becomes directly inspectable instead of inferred from
scattered current-state files.

#### Details

The implementation should centralize event append behavior in a small shared
writer/helper rather than duplicating JSONL formatting across lifecycle,
review, and evidence packages. Each event should include stable ordering data,
command identity, concise summary text, revision, artifact references, and
when applicable a `from_node` to `to_node` transition hint. The event should
reference existing artifacts rather than embedding full review/evidence bodies.

This step must avoid compatibility scaffolding for old worktrees. New command
executions should produce authoritative timeline entries immediately, and tests
should focus on correctness of newly written data rather than migration logic.

#### Expected Files

- new timeline/event helper package(s) under `internal/`
- `internal/lifecycle/service.go`
- `internal/review/service.go`
- `internal/evidence/service.go`
- `internal/runstate/` files if the shared append helper belongs there
- new or updated tests under `internal/`

#### Validation

- Command success paths append the expected event line for the active plan.
- The appended events preserve enough information to reconstruct ordered
  timeline entries for lifecycle, review, and evidence operations.
- Regression tests cover multiple event kinds and confirm references point at
  the correct existing artifacts.

#### Execution Notes

Added a new `internal/timeline/` package that owns the append-only event index
path, line parsing, and `/api/timeline` read model. Event appends now happen in
the approved mutating workflow commands by taking pre-command snapshots,
computing the post-command state while the service still owns its mutation
lock, and writing one normalized event line with summary text, revision,
transition context, and artifact refs into
`.local/harness/plans/<plan-stem>/events.jsonl` before the command returns
success. The first finalize review surfaced two correctness issues in the
original append strategy: event order could drift under concurrent commands,
and a command could report failure after its mutation had already committed if
timeline append failed. The implementation was tightened so lifecycle, review,
and evidence services now treat timeline append as part of the successful local
transaction boundary and roll back their just-written runtime artifacts if the
event cannot be recorded. Focused CLI coverage now asserts event emission for
`review submit`, `review aggregate`, `archive`, `reopen`, `land`, and
`land complete` in addition to the earlier `execute start`, `review start`, and
`evidence submit` checks, with rollback regression tests for failed append
paths. A second finalize review also pushed the event writer from direct
append-mode writes to a crash-safe atomic rewrite under the plan-local timeline
lock so interrupted writes cannot leave a truncated JSONL tail that breaks
timeline loading for the plan. A later finalize full review caught one more
transaction-boundary bug in `archive` and `reopen`: the success event could be
recorded before the source plan file was removed. The lifecycle service now
removes the source plan before calling the post-mutation timeline hook, and
`rollbackTransition` restores the original plan path if a late timeline append
failure happens after that cleanup step.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this event-append step was developed as part of one
integrated timeline slice and will receive a full finalize review before
archive.

### Step 3: Expose a timeline resource and render the live Timeline page

- Done: [x]

#### Objective

Replace the UI placeholder with a real read-only timeline view backed by the
new event index and a thin Go resource/view model.

#### Details

The Go UI layer should expose a dedicated timeline endpoint or resource for the
current worktree. The frontend should consume that resource, render ordered
entries with timestamps, command/kind labels, summaries, transition context,
and high-signal artifact references, and keep the page read-only. The UI does
not need to become a full artifact document browser in this slice, but the
rendered data should make the recent trajectory legible without forcing the
human to reconstruct it from raw files.

The implementation should update focused browser automation so `Timeline` is no
longer validated as a generic WIP placeholder. In addition to scripted checks,
this step should include a manual step-by-step browser pass using the
[$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill to
verify navigation, real timeline rendering, and the visual presentation of the
page. That manual pass should capture screenshots or equivalent browser
artifacts so the resulting UI state is reviewable after execution.

#### Expected Files

- `internal/ui/server.go`
- `internal/ui/server_test.go`
- `web/src/main.tsx`
- `web/src/styles.css`
- `scripts/ui-playwright-smoke`
- any new Go-side timeline view-model/resource files under `internal/ui/`

#### Validation

- `GET` of the new timeline resource succeeds for representative local states.
- The `Timeline` page renders real entries and no longer advertises itself as
  a generic WIP page.
- Browser validation covers route load plus at least one realistic populated
  timeline case.
- A manual Playwright validation pass captures screenshots that verify the live
  `Timeline` page's functionality and visual presentation.

#### Execution Notes

Added `GET /api/timeline` to the Go UI server and replaced the `Timeline`
placeholder with a real read-only view that fetches and renders live event
data. The final UI layout now uses the intended three-column workbench shape:
the left rail stays narrow, the middle column is the event-navigation list,
and the right column is the raw payload editor pane with `Event`/`Input`/
`Output`/`Artifacts` tabs when those payloads exist. Updated the browser smoke
flow to validate the live Timeline route instead of the old WIP text and to
seed a representative isolated runtime event when needed for deterministic
browser validation. Validation covered `go test ./...`, `pnpm --dir web check`,
`pnpm --dir web build`, `scripts/sync-contract-artifacts --check`, and
`bash scripts/ui-playwright-smoke`, plus manual Playwright passes with
captured screenshots under `output/playwright/manual-timeline/` and
`output/playwright/manual-timeline-redesign/`. Finalize review follow-up also
flagged that timeline loading still relied on `bufio.Scanner`'s 1 MiB token
cap even though timeline events now carry raw input/output/artifact payloads,
so the reader now streams `events.jsonl` line-by-line and a regression test
covers a `>1 MiB` timeline event payload. The timeline reader still degrades
gracefully when the local state cache is absent or unreadable and continues to
serve the event index with a soft error entry instead of failing the resource
outright. A later finalize full review also tightened the validation surface:
the UI smoke flow now clicks into the live raw payload inspector and verifies
the `Output` tab rendering, the UI README entry no longer describes `Timeline`
as a WIP placeholder, and `/api/timeline` coverage now proves that a 2 MiB
event payload survives API serialization without truncation. A final UX pass
then simplified the middle explorer density, made the event list scroll
independently from the editor pane, removed redundant right-pane summary/footer
chrome, moved transition context into the event header, and turned
`artifact_refs` into one raw tab per referenced artifact so the detail pane
behaves more like a file-tab workbench than a stack of cards. The latest
finalize-fix pass then addressed the remaining `review-009-full` findings:
timeline default selection now opens on the most recent non-synthetic event
instead of the oldest bootstrap row, `review start` / `review aggregate` /
`evidence submit` now roll back newly written durable artifacts when local
state persistence fails, and the browser smoke flow now seeds a deterministic
archive-to-land fixture so the UI validation proves raw handoff inspection for
`archive`, `evidence submit`, and `land`. Manual Playwright verification also
captured fresh screenshots under `output/playwright/manual-timeline-final/`
for the current default view and a `review start` artifact-tab inspection. The
latest reopen then tightened the remaining presentation polish: the middle
event navigator now renders newest-first so fresh activity stays at the top,
and file-backed tabs now read like content tabs (`Manifest`, `Submission`,
`Publish Record`) instead of exposing raw `*_path` implementation names.
Validation for that pass refreshed embedded UI assets, reran
`go test ./...`, reran `bash scripts/ui-playwright-smoke`, and captured new
headed Playwright screenshots under `output/playwright/manual-timeline-r3/`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this UI/resource step was developed as part of one
integrated timeline slice and will receive a full finalize review before
archive.

## Validation Strategy

- Run focused Go tests for the touched contract, command, and UI packages.
- Run `scripts/sync-contract-artifacts` and confirm any generated contract
  artifacts are in sync.
- Run `harness plan lint` on this tracked plan before execution starts.
- Run focused browser checks for the live `Timeline` route and complete a
  manual step-by-step Playwright validation pass with screenshots/artifacts for
  visual and functional review.

## Risks

- Risk: The event index could drift into a second competing state model.
  - Mitigation: keep event payloads minimal, reference existing artifacts, and
    treat `harness status` as the only canonical current-state resolver.
- Risk: Command-by-command event append logic could become inconsistent.
  - Mitigation: centralize append behavior and cover representative event kinds
    with focused regression tests.
- Risk: The first UI view may overfit to one event family and hide others.
  - Mitigation: define one normalized timeline view model before wiring the
    page so lifecycle, review, and evidence entries all render through the same
    presentation contract.

## Validation Summary

UPDATE_REQUIRED_AFTER_REOPEN

The original archive candidate was reopened in `finalize-fix` mode after
human feedback narrowed one remaining Timeline semantic gap: the right-hand
inspector still rendered artifact-ref tabs as ref metadata instead of file
contents, and it still exposed context-only tabs such as `plan_path`. The
current revision 3 follow-up then closed the last two presentation gaps:
path-backed file tabs still showed raw labels like `submission_path`, and the
event explorer still rendered oldest-first instead of showing the freshest
activity at the top.

- `go test ./...` passes after the timeline transaction-boundary fixes,
  including new internal rollback coverage for `review start`,
  `review submit`, `review aggregate`, and `evidence submit` (`ci`,
  `publish`, and `sync`).
- `go test ./internal/timeline ./internal/ui` also covers the new read-model
  artifact-content expansion paths, including resolved file-content payloads
  on `/api/timeline`.
- `pnpm --dir web check` and `pnpm --dir web build` pass with the final
  three-column Timeline layout, refined tab filtering, and refreshed embedded
  assets.
- `bash scripts/ui-playwright-smoke` passes with a deterministic
  archive-to-land fixture that validates the live Timeline route, default
  latest-event selection, raw `Output` inspection, resolved artifact-content
  tabs, hidden context-only tabs, lock contention handling, and Vite dev
  rendering.
- Manual Playwright screenshots captured the live UI under
  `output/playwright/manual-timeline-final/timeline-default-current-state.png`
  and
  `output/playwright/manual-timeline-final/timeline-review-start-manifest-tab.png`.
- Additional finalize-fix screenshots now live under
  `output/playwright/manual-timeline-artifact-content/`, including
  `timeline-review-submit-output.png` and
  `timeline-review-start-manifest-content.png`.
- A revision 3 follow-up pass also refreshed the embedded UI assets, reran
  `go test ./...`, reran `bash scripts/ui-playwright-smoke`, and captured
  headed Playwright screenshots under `output/playwright/manual-timeline-r3/`,
  including `review-start-manifest.png` and
  `review-submit-submission.png`.

## Review Summary

UPDATE_REQUIRED_AFTER_REOPEN

The original archive candidate passed `review-011-full`, then reopened for one
narrow finalize follow-up after direct human UX feedback on the Timeline
inspector tabs.

- `review-006-full` and `review-007-delta` drove the transaction-boundary
  rollback fixes that keep timeline events aligned with successful command
  mutations.
- `review-008-delta` passed after the nil-state reopen rollback fix.
- `review-009-full` found four blockers: state-save orphan artifacts in
  `review start` / `evidence submit`, default selection opening on bootstrap,
  and smoke coverage missing the archive/land handoff path.
- `review-010-full` found the remaining early-stage persistence gaps in
  `review start` / `review submit`, missing rollback coverage for
  `publish` / `sync`, and placeholder closeout sections in this tracked plan.
- `review-011-full` passed with zero blocking and zero non-blocking findings
  after the final rollback fixes, coverage additions, smoke refresh, and
  closeout-summary updates.
- `review-012-delta` passed with zero blocking and zero non-blocking findings
  after the finalize-fix follow-up for resolved artifact-content tabs, hidden
  context-only tabs, and the refined Timeline inspector behavior after reopen.
- `review-013-delta` passed with zero blocking and zero non-blocking findings
  at `2026-04-02T22:14:36+08:00` after the revision 3 polish pass for
  newest-first explorer ordering and content-oriented artifact-tab labels.

## Archive Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Archived At: 2026-04-02T22:15:49+08:00
- Revision: 3
This section records the first archive handoff and the state of the reopened
candidate. Revision 1 archived successfully, then revision 2 reopened in
`finalize-fix` mode after new UI feedback invalidated merge-readiness. The
candidate has since reopened again into revision 3 for one last Timeline
presentation polish pass.

- Finalize Review: `review-013-delta` passed at
  `2026-04-02T22:14:36+08:00`.
- PR: https://github.com/catu-ai/easyharness/pull/102 remains the working PR,
  but it is not merge-ready again until the revision 3 archive is committed,
  pushed, and backed by refreshed publish/CI/sync evidence.
- Ready: yes for archive closeout; `review-013-delta` passed clean and the
  remaining work is the controller-owned archive/publish/CI/sync loop.
- Merge Handoff: commit and push the revision 3 archive plus UI polish,
  recompute publish/CI/sync evidence, and then wait for human merge approval.

## Outcome Summary

### Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- Added the command-owned `.local/harness/plans/<plan-stem>/events.jsonl`
  contract plus the `/api/timeline` read model and UI wiring.
- Replaced the Timeline placeholder with a VS Code-like three-column workbench:
  narrow rail, independently scrolling event navigator, and raw detail pane
  with payload/artifact tabs.
- Refined the Timeline inspector so `Event`, `Input`, and `Output` stay
  separate, path-backed artifact tabs render referenced file contents, and
  context-only refs such as `plan_path` / `local_state_path` stay out of the
  tab strip.
- Polished the follow-up Timeline UX so the event explorer now shows newest
  activity first and file-backed tabs are labeled by the content being viewed
  (`Manifest`, `Submission`, `Publish Record`) instead of raw `*_path`
  identifiers.
- Hardened lifecycle, review, and evidence timeline writes so successful
  command results roll back if later timeline/state persistence fails.
- Added focused regression coverage for large timeline payloads and rollback
  paths, plus scripted and manual browser validation for the live Timeline
  route.

### Not Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- The archive move itself, publish evidence refresh, CI/sync evidence refresh,
  and renewed merge-ready handoff for revision 3 are still pending the
  post-review archive/publish loop.
- Rich inline artifact viewers beyond raw JSON/tabbed inspection remain
  deferred.

### Follow-Up Issues

UPDATE_REQUIRED_AFTER_REOPEN

- #101: Track post-launch timeline follow-ups after event-index landing

