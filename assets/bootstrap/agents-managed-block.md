## Harness Working Agreement

1. Humans steer. Agents execute.
2. Approved scope lives in a git-tracked plan.
3. Raw execution trajectory lives in `.local/` and is disposable.
4. Durable summaries, contracts, and behavior changes must be written back to
   tracked docs or code before archive.
5. Evidence beats memory. Use `harness status`, tracked plans, and owned local
   artifacts instead of relying on long-session recall.
6. Keep tracked docs and code in English.

## Harness Source of Truth

The default harness split in this repository is:

- tracked plan in `docs/plans/`: scope, durable step closeout, archive-ready summaries
- `.local/harness/plans/archived/`: archived lightweight plan snapshots
- `.local/harness/`: disposable runtime state, review artifacts, evidence artifacts, and trajectory
- `docs/specs/`: normative harness contracts
- `.agents/skills/`: repo-local harness workflow skills

If a tracked plan conflicts with a repo-local skill, the tracked plan wins.

## Harness Workflow

For medium or large work:

1. Discovery
2. Plan
3. Execute
4. Archive / publish / await merge approval
5. Land

For approved low-risk work that explicitly uses `workflow_profile:
lightweight`, keep the same workflow shape but store the active plan under
`docs/plans/active/` like any other plan. Only the archived lightweight
snapshot moves to `.local/harness/plans/archived/<plan-stem>.md`. That
shortcut does not remove human steering, low-risk eligibility checks, or the
requirement to leave a repo-visible breadcrumb such as a PR body note.

Use `lightweight` only when all of these are true:

- the whole slice is one bounded low-risk maintenance change
- the edits are limited to README/docs/comments/copy or similarly
  non-behavioral cleanup
- no `harness` behavior, normative spec, state rule, persistence behavior,
  release or CI workflow, or security-sensitive logic changes
- if the boundary is unclear, default to `standard`

Use `harness reopen --mode finalize-fix|new-step` when an archived candidate
is no longer merge-ready because of new feedback, remote changes, or other
invalidation.

## Harness Review Execution

When work enters review orchestration, spawned reviewer subagents are the
default path. The controller agent stays in `harness-execute`, reviewer work
belongs to spawned `harness-reviewer` subagents, and the repo-local review
skills must be followed strictly.

Routine review progression is controller-owned once a tracked plan is approved.
The controller should not stop to ask the human whether ordinary step-closeout
or finalize review should begin.

Use `harness status` at routine checkpoints:

- when starting or resuming execution
- before marking a step done
- after each review aggregate
- before relying on later-step or finalize progression after a warning or fix

Human confirmation is still required for real blockers, scope changes, and
merge approval, but not for ordinary review closeout.

If an approved plan is likely to require reviewer subagents later, ask for
explicit human authorization when seeking plan approval instead of waiting
until review orchestration is already blocked on that permission.

## Harness Start Points

When entering the repository or resuming after compaction:

1. Read `README.md` if you need repository purpose or setup context.
2. Run `harness status`.
3. If `harness status` reports a current plan artifact, open that plan.
   Active work always uses a tracked plan under `docs/plans/active/`; archived
   lightweight candidates may live under `.local/harness/plans/archived/`.
   If status reports `idle`, there is no current plan to resume yet.
4. Most resumed work should continue in `harness-execute`.
5. Switch only when `harness status` and the workflow boundary clearly call for
   a different skill:
   - `harness-discovery` when direction is unclear
   - `harness-plan` when creating or revising a tracked plan
   - `harness-land` only when `state.current_node` is
     `execution/finalize/await_merge` and a human has explicitly approved
     merge
   - `harness-reviewer` only inside spawned reviewer subagents
