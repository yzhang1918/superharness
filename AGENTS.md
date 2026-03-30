# AGENTS.md

This document defines how humans and Codex collaborate in `easyharness`.

## Mission

Build `easyharness` as a thin, git-native, agent-first harness system that is
easier to understand and maintain than a scripts-heavy workflow. The project
name is `easyharness`; the CLI executable remains `harness`.

## Working Agreement

1. Humans steer. Agents execute.
2. Approved scope lives in a git-tracked plan.
3. Raw execution trajectory lives in `.local/` and is disposable.
4. Durable summaries, contracts, and behavior changes must be written back to
   tracked docs or code before archive.
5. Evidence beats memory. Use `harness status`, tracked plans, and owned local
   artifacts instead of relying on long-session recall.
6. Keep tracked docs and code in English.

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

## Source of Truth

The source-of-truth split in this repository is:

- tracked plan in `docs/plans/`: scope, durable step closeout, archive-ready summaries
- specs in `docs/specs/`: normative harness contracts
- `.local/harness/`: disposable runtime state, review artifacts, evidence
  artifacts, and trajectory
- skills in `.agents/skills/`: how Codex should operate inside those contracts

If a skill conflicts with a tracked spec or approved plan, the spec or plan
wins.

## Required Workflow

For medium or large work:

1. Discovery
2. Plan
3. Execute
4. Archive / publish / await merge approval
5. Land

Use `harness reopen --mode finalize-fix|new-step` when an archived candidate
is no longer merge-ready because of new feedback, remote changes, or other
invalidation.

## Review Execution

When work enters review orchestration, spawned reviewer subagents are the
default and required path. The controller agent stays in `harness-execute`,
reviewer work belongs to spawned `harness-reviewer` subagents, and the
repo-local review skills must be followed strictly.

Routine review progression is controller-owned once a tracked plan is approved.
The controller must not stop to ask the human whether ordinary step-closeout or
finalize review should begin.

Use `harness status` at routine checkpoints:

- when starting or resuming execution
- before marking a step done
- after each review aggregate
- before relying on later-step or finalize progression after a warning or fix

Human confirmation is still required for real blockers, scope changes, and
merge approval, but not for ordinary review closeout.

If an approved plan is likely to require reviewer subagents later, ask for
explicit human authorization to spawn them when seeking plan approval instead
of waiting until review orchestration is already blocked on that permission.
If execution still reaches a reviewer-subagent boundary without that approval,
pause only long enough to request it explicitly, then continue once the human
answers.

## Start Points

When entering the repo or resuming after compaction:

1. Read [README.md](./README.md) if you need repository purpose or setup
   context.
2. Run `harness status`.
3. If `harness status` reports a current plan artifact, open that tracked
   plan. If status reports `idle`, there is no current plan to resume yet.
4. Most resumed work should continue in `harness-execute`.
5. Switch only when `harness status` and the workflow boundary clearly call
   for a different skill:
   - `harness-discovery` when direction is unclear
   - `harness-plan` when creating or revising a tracked plan
   - `harness-land` only when `state.current_node` is
     `execution/finalize/await_merge` and a human has explicitly approved
     merge
   - `harness-reviewer` only inside spawned reviewer subagents

## Git and PR Rules

- main branch: `main`
- working branches: `codex/<topic>`
- commits: small and reviewable
- append `Co-authored-by: Codex <codex@openai.com>` unless the human requests
  otherwise
- when writing multi-line git or gh bodies, prefer heredocs so shell quoting
  does not eat backticks or other structured text
- default merge strategy: `Merge commit`
- do not rewrite shared history without explicit approval

If work creates durable deferred scope, create or update GitHub issues before
archive and record them in the plan.
