# AGENTS.md

This document defines how humans and Codex collaborate in `superharness`.

## Mission

Build `superharness` as a thin, agent-first harness system that is easier to
understand and maintain than a scripts-heavy workflow.

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

- tracked plan in `docs/plans/`: scope, lifecycle, archive-ready summaries
- specs in `docs/specs/`: normative harness contracts
- `.local/harness/`: disposable runtime state, review artifacts, CI snapshots,
  sync snapshots, and trajectory
- skills in `.agents/skills/`: how Codex should operate inside those contracts

If a skill conflicts with a tracked spec or approved plan, the spec or plan
wins.

## Required Workflow

For medium or large work:

1. Discovery
2. Plan
3. Execute
4. Archive / await merge approval
5. Land

Use `reopen` when an archived candidate is no longer merge-ready because of
new feedback, remote changes, or other invalidation.

## Start Points

When entering the repo or resuming after compaction:

1. Read [README.md](./README.md) if you need repository purpose or setup
   context.
2. Run `harness status`.
3. Open the current tracked plan named by `harness status`.
4. Most resumed work should continue in `harness-execute`.
5. Switch only when the lifecycle clearly calls for a different skill:
   - `harness-discovery` when direction is unclear
   - `harness-plan` when creating or revising a tracked plan
   - `harness-land` only after explicit human merge approval
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
