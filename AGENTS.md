# AGENTS.md

This document defines repo-specific guidance for how humans and Codex
collaborate in `easyharness`.

## Mission

Build `easyharness` as a thin, git-native, agent-first harness system that is
easier to understand and maintain than a scripts-heavy workflow. The project
name is `easyharness`; the CLI executable remains `harness`.

## Development Prerequisite

Before using repo-local skills that call `harness`, make sure the command is
available:

```bash
command -v harness
```

If not, bootstrap it from this repository:

```bash
scripts/install-dev-harness
```

If you change Go CLI code, rerun the installer before relying on the direct
`harness` command again.

## Bootstrap Asset Editing

This repository dogsfoods the same bootstrap assets that `harness install`
packages for other repositories.

- Edit `assets/bootstrap/` when changing the harness-managed skill pack or the
  managed `AGENTS.md` block content.
- Treat `.agents/skills/` in this repository as tracked materialized output from
  `assets/bootstrap/`, not as a hand-edited source tree.
- After editing `assets/bootstrap/`, run `scripts/sync-bootstrap-assets` to
  refresh `.agents/skills/` and the managed block in this root `AGENTS.md`.
- Keep easyharness-specific guidance in this root `AGENTS.md` outside the
  managed markers below.

The block below is the same harness-managed repository contract that
`harness install --scope agents` would install into another repository.
Keep easyharness-specific guidance outside the managed markers.

<!-- easyharness:begin -->
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

## Harness Subagent Use

The controller owns shared repository context and the final workflow judgment.
Spawn subagents only for bounded subproblems; do not split one shared context
bundle across multiple subagents just to get summaries back.

Discovery and execution may stay local, use one subagent, or use multiple
subagents in parallel according to the current question shape:

- stay local when the controller can answer the next question from the shared
  context it already needs to hold
- use `1` when one bounded question or hypothesis needs independent repo
  checking
- use multiple subagents in parallel only when multiple hypotheses or
  questions are genuinely independent

In Codex, spawned subagents are not fire-and-forget memory. Once a bounded
subagent task is complete and the controller has received the result, close
that subagent promptly by default. Reuse `resume_agent` only when a later
narrow follow-up makes continuity materially more valuable than a fresh agent.

## Harness Review Execution

When work enters review orchestration, spawned reviewer subagents are the
default path. The controller agent stays in `harness-execute`, reviewer work
belongs to spawned `harness-reviewer` subagents, and the repo-local review
skills must be followed strictly. The shared rules in `Harness Subagent Use`
still apply here; review-specific docs add reviewer-slot orchestration,
aggregation, and same-slot resume rules on top of that shared baseline.

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
<!-- easyharness:end -->

## Git and PR Rules

- main branch: `main`
- working branches: `codex/<topic>`
- commits: small and reviewable
- append `Co-authored-by: Codex <codex@openai.com>` unless the human requests
  otherwise
- when writing multi-line git or gh bodies, prefer heredocs so shell quoting
  does not eat backticks or other structured text
- when using the lightweight workflow, leave the agreed repo-visible breadcrumb
  in the PR body or other approved review surface before treating the candidate
  as ready to wait for merge approval
- default merge strategy: `Merge commit`
- do not rewrite shared history without explicit approval

If work creates durable deferred scope, create or update GitHub issues before
archive and record them in the plan.
