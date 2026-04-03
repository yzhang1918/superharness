---
template_version: 0.2.0
created_at: "2026-04-03T23:05:10+08:00"
source_type: issue
source_refs:
    - '#88'
---

# Define the discovery explorer-subagent contract

## Goal

Define how `harness-discovery` should use on-demand explorer subagents without
turning repository reading into a mandatory or opaque side workflow. The
result should make discovery predictable for future agents by distinguishing
controller-owned shared context from hypothesis-driven exploration, and by
recording when `0`, `1`, or multiple explorer subagents are appropriate.

This slice should also make the Codex-specific subagent lifecycle explicit at
the repository-contract level so discovery and review share the same default
expectation: once a bounded subagent task is done and its result is received,
close that agent promptly unless a later narrow follow-up makes `resume_agent`
materially helpful.

## Scope

### In Scope

- Define a repository-visible subagent-use contract that applies across
  workflows and names Codex lifecycle expectations explicitly.
- Update the bootstrap `harness-discovery` skill to describe when discovery
  should stay local, when it may spawn one explorer, and when multiple
  independent hypotheses justify parallel explorers.
- Record the discovery explorer output contract: explorers return factual
  bounded context only, while the controller keeps shared context ownership and
  decides the next human question.
- Sync bootstrap assets into the materialized repo-local skill tree and the
  managed `AGENTS.md` block.

### Out of Scope

- Changing execution or review orchestration behavior beyond wording alignment
  needed for the shared Codex subagent lifecycle rule.
- Introducing new CLI commands, harness state fields, or automated review
  orchestration for discovery.
- Defining agent-agnostic lifecycle guarantees for coding agents other than
  Codex.

## Acceptance Criteria

- [x] `AGENTS.md` contains a shared subagent-use section that states controller
      ownership of shared context, allows `0`/`1`/multiple subagents based on
      bounded independent hypotheses, and explicitly names Codex's default
      close-after-use lifecycle.
- [x] The bootstrap `harness-discovery` skill explains that user-supplied core
      context should normally stay with the controller, and that explorer use
      is demand-driven rather than mandatory.
- [x] The discovery contract explains that multiple explorers are appropriate
      only for independent hypotheses or questions, not for splitting one
      shared context bundle into summaries.
- [x] The discovery contract explains that explorer outputs are factual
      context reports only and do not choose the next question or workflow
      direction.
- [x] `scripts/sync-bootstrap-assets` refreshes the materialized skill and
      managed `AGENTS.md` content, and the resulting tracked plan lints cleanly.

## Deferred Items

- Any future decision to generalize lifecycle wording for non-Codex agent
  runtimes.
- Additional discovery tooling, templates, or prompt scaffolds beyond the
  contract language update.
- Refactoring review reference docs unless execution uncovers a concrete drift
  or contradiction with the new shared subagent wording.

## Work Breakdown

### Step 1: Add a shared subagent-use contract to the managed repo guidance

- Done: [x]

#### Objective

Add a reusable harness-level contract in the managed `AGENTS.md` block so
discovery and review share the same top-level subagent rules.

#### Details

The new section should sit between the general workflow description and the
review-specific execution rules. It should say that the controller owns shared
repository context and final workflow judgment, while spawned subagents are for
bounded subproblems only. It should explicitly allow `0`, `1`, or multiple
subagents when the current work presents no hypothesis split, one bounded
question, or multiple independent hypotheses respectively. It should also name
Codex specifically: `spawn_agent` is not fire-and-forget memory, so completed
subagents should be closed promptly after their bounded task is consumed, with
`resume_agent` reserved for later narrow follow-up where continuity matters.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `AGENTS.md`

#### Validation

- The managed block reads as a workflow-agnostic contract rather than a
  discovery-only or review-only note.
- The Codex lifecycle wording does not conflict with the more detailed review
  reference guidance.

#### Execution Notes

Added a new `Harness Subagent Use` section to the managed bootstrap contract in
`assets/bootstrap/agents-managed-block.md`. The section stays principle-level:
the controller owns shared context and final workflow judgment; `0`, `1`, or
multiple subagents are all valid depending on whether the open work presents no
hypothesis split, one bounded repository question, or multiple independent
hypotheses; and Codex-specific lifecycle guidance now says completed bounded
subagents should be closed promptly by default, with `resume_agent` reserved
for later narrow follow-up where continuity matters.

Ran `scripts/sync-bootstrap-assets`, which refreshed the managed block in the
root `AGENTS.md` with the same wording.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this shared contract change is narrow, documented in the
tracked plan, and will be covered by branch-level closeout review.

### Step 2: Teach `harness-discovery` when and how to use explorer subagents

- Done: [x]

#### Objective

Update the discovery bootstrap skill so future controllers can use bounded
explorer subagents intentionally without losing shared context ownership.

#### Details

The skill should describe discovery as an iterative loop where exploration may
occur both before and after a human clarification answer, always on demand.
The controller should keep user-supplied core context local. Explorer use
should be driven by the number and independence of the hypotheses that still
need verification, not by raw file-count or detail volume. One explorer fits a
single bounded unknown; parallel explorers fit multiple independent unknowns.
Explorers should report only factual findings relevant to the bounded question,
leaving interpretation, option framing, and the next human question to the
controller.

#### Expected Files

- `assets/bootstrap/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-discovery/SKILL.md`

#### Validation

- The skill gives a future controller enough guidance to choose between local
  reading, one explorer, or multiple explorers without relying on discovery
  chat memory.
- The skill clearly separates factual explorer results from controller-owned
  questioning and decision-making.

#### Execution Notes

Updated the bootstrap `harness-discovery` skill so exploration is explicitly
demand-driven. The controller now keeps user-supplied core context and other
shared repository context locally when later questioning may depend on the
details; discovery may choose local reading, one explorer, or multiple
parallel explorers based on the number and independence of bounded hypotheses;
and explorer subagents are limited to factual findings rather than workflow or
question-selection judgment.

The synced materialized skill under `.agents/skills/harness-discovery/SKILL.md`
now matches the bootstrap source.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the discovery-skill contract update is part of the same
integrated documentation slice and will be validated again during closeout.

### Step 3: Sync the bootstrap outputs and verify the contract stays coherent

- Done: [x]

#### Objective

Refresh the materialized repo outputs and verify the new discovery contract is
internally consistent.

#### Details

After editing the bootstrap sources, run `scripts/sync-bootstrap-assets` so the
tracked materialized skill tree and managed `AGENTS.md` block reflect the new
contract. Check whether review-specific docs need wording alignment; only make
minimal edits if the shared Codex lifecycle section would otherwise contradict
the existing review reference. Keep the final result narrow and contract-
focused.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `assets/bootstrap/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-discovery/SKILL.md`
- `AGENTS.md`
- any review reference file only if alignment is required

#### Validation

- Run `scripts/sync-bootstrap-assets`.
- Run `harness plan lint docs/plans/active/2026-04-03-discovery-explorer-subagent-contract.md`.
- Reread the synced discovery skill and managed `AGENTS.md` block for contract
  consistency.

#### Execution Notes

Ran `scripts/sync-bootstrap-assets`, which reported `0` files created and `2`
files updated, refreshing the root managed `AGENTS.md` block and the
materialized `.agents/skills/harness-discovery/SKILL.md` output. Re-read the
synced files alongside the bootstrap sources to confirm the shared subagent
rules and discovery-specific explorer rules stay aligned.

No review-reference edits were needed: the new `AGENTS.md` section stays at the
principle level and does not contradict the existing detailed review
orchestration guidance around reviewer lifecycle and aggregation.

#### Review Notes

NO_STEP_REVIEW_NEEDED: sync verification is mechanical and the overall slice
will receive branch-level review.

## Validation Strategy

- Use `harness plan lint` to validate the tracked plan structure before asking
  for approval.
- During execution, use a direct reread of the synced bootstrap outputs to
  verify that shared subagent rules and discovery-specific explorer rules stay
  aligned.
- Re-run `harness plan lint` after step completion so the tracked plan remains
  structurally valid with execution notes filled in.

## Risks

- Risk: The new shared subagent wording could duplicate or conflict with the
  detailed review-orchestration guidance.
  - Mitigation: Keep `AGENTS.md` at the principle level and leave
    review-orchestration specifics in the review reference unless a direct
    contradiction appears.
- Risk: Discovery wording could accidentally encourage over-delegation and
  hidden planning.
  - Mitigation: Explicitly anchor explorer use to bounded independent
    hypotheses, keep user-supplied shared context with the controller, and
    restrict explorer outputs to factual reports.

## Validation Summary

- `scripts/install-dev-harness`
- `harness plan lint docs/plans/active/2026-04-03-discovery-explorer-subagent-contract.md`
- `scripts/sync-bootstrap-assets`
- `harness plan lint docs/plans/active/2026-04-03-discovery-explorer-subagent-contract.md`
- direct reread of the synced `AGENTS.md` block and
  `.agents/skills/harness-discovery/SKILL.md` to confirm they match the
  bootstrap sources and remain consistent with the existing review reference
- direct reread after revision 2 to confirm the shared subagent section now
  uses `stay local` wording instead of the earlier `0 subagents` phrasing
- direct reread after revision 3 to confirm review orchestration explicitly
  inherits the shared Codex lifecycle rules and discovery now isolates
  explorer-subagent choice into its own reusable decision section

## Review Summary

- `review-001-full`: finalize review passed with no findings across the
  `docs_consistency` and `agent_ux` dimensions
- `review-002-delta`: reopen follow-up review passed with no findings after the
  wording fix replaced the rigid `0 subagents` phrasing with `stay local`
- `review-003-delta`: reopen follow-up review passed with no findings after
  making the review side explicitly inherit the shared subagent rules and
  moving discovery explorer policy into a standalone decision section

## Archive Summary

- Archived At: 2026-04-03T23:41:12+08:00
- Revision: 3
- PR: https://github.com/catu-ai/easyharness/pull/106
- Ready: The candidate satisfies the acceptance criteria, the managed
  `AGENTS.md` contract now defines shared subagent use plus the Codex
  close-after-use default, review orchestration now explicitly inherits those
  shared rules, discovery now exposes explorer use as a standalone decision
  module, and `review-003-delta` passed clean.
- Merge Handoff: Re-archive the candidate, commit the revision-3 modularity and
  shared-review-rule fixes on `codex/discovery-explorer-subagents`, push the
  branch to update PR #106, and refresh publish/CI/sync evidence until
  `harness status` returns to
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added a shared `Harness Subagent Use` section to the managed bootstrap
  `AGENTS.md` contract and synced it into the repository root managed block.
- Taught `harness-discovery` to treat explorer use as demand-driven, preserve
  controller ownership of shared context, and allow `0`, `1`, or multiple
  explorers based on bounded independent hypotheses.
- Constrained discovery explorers to factual bounded outputs so controller
  synthesis and next-question decisions remain local.
- Refreshed the materialized `.agents` discovery skill output and verified the
  new wording stays aligned with the existing review orchestration reference.
- Refined the shared subagent section wording so the local case now reads
  `stay local` instead of the more mechanical `0 subagents` phrasing.
- Made `Harness Review Execution` explicitly point back to the shared
  subagent-use rules, and updated the review-orchestration reference so its
  Codex section is clearly an extension of that shared baseline.
- Refactored discovery so explorer-subagent policy lives in a dedicated
  `Explorer Subagent Decision` section instead of being embedded inline in the
  execution flow.

### Not Delivered

- No new CLI or harness-state support was added for discovery orchestration.
- No generalized lifecycle contract was added for non-Codex agent runtimes.
- The review reference still keeps reviewer-specific lifecycle details locally
  instead of extracting them into a separate shared reference file.

### Follow-Up Issues

No new follow-up issue was created in this slice. The deferred items remain
backlog candidates for future agent-runtime generalization or broader discovery
tooling work.
