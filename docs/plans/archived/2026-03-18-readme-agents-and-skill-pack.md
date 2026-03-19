---
status: archived
lifecycle: awaiting_merge_approval
revision: 3
template_version: 0.1.0
created_at: "2026-03-18T22:25:00+08:00"
updated_at: "2026-03-19T23:26:43+08:00"
source_type: issue
source_refs:
    - '#5'
    - https://github.com/yzhang1918/superharness/pull/10
---

# Bootstrap README, AGENTS, and skill pack

## Goal

Turn `superharness` from a repository that merely contains a CLI into a
repository that can explain and drive its own workflow. This slice should make
the repo legible to both humans and agents by adding a human-facing README, an
agent-facing `AGENTS.md`, and the first repo-local skill pack built around the
existing harness lifecycle.

The outcome should be strong enough that a fresh Codex session can enter the
repo, learn how work is supposed to flow, invoke `harness` directly in the
current development environment, and dogfood the repository using the same
contracts it is trying to establish.

## Scope

### In Scope

- Add a `README.md` that explains what `superharness` is, which commands exist
  today, how the workflow is intended to operate, and how to use the repo in
  development.
- Add an `AGENTS.md` that records the repository's human/agent working
  agreement and points execution detail to the new skill pack.
- Add the first repo-local skill pack with exactly five top-level skills:
  `harness-discovery`, `harness-plan`, `harness-execute`, `harness-land`, and
  `harness-reviewer`.
- Organize `harness-execute` with references instead of proliferating more top-level
  loop skills.
- Make `harness` invocable as a direct command in this development environment
  without requiring users or skills to spell `go run ./cmd/harness ...`.
- Encode the current review orchestration rule in the skills:
  one active review round at a time, wait for every spawned reviewer subagent
  to finish before aggregating, and explicitly close reviewer subagents after
  their results are consumed.
- Dogfood the new docs and skills against this repo.

### Out of Scope

- `harness ui` and any web UI implementation.
- Formal release packaging, Homebrew publishing, or installer support beyond
  development-time setup for this repository.
- Supporting overlapping active review rounds in v0.1/v0.2 skills.
- Broad test-fixture infrastructure beyond the tests or validation directly
  needed for this slice.

## Acceptance Criteria

- [x] The repository contains a `README.md` that explains the project,
      development-time setup, current command surface, and current workflow in
      a way that a new human collaborator can follow.
- [x] The repository contains an `AGENTS.md` that defines the repo-level
      working agreement, source-of-truth split, and lifecycle expectations for
      Codex agents in this repo.
- [x] The repository contains repo-local skills for `harness-discovery`,
      `harness-plan`, `harness-execute`, `harness-land`, and
      `harness-reviewer`, with `harness-execute` decomposed through
      references rather than more top-level loop skills.
- [x] The `harness-execute` and `harness-reviewer` skill contracts explicitly
      document the
      current Codex-specific reviewer orchestration rules:
      wait until all reviewer subagents finish before aggregation, and close
      reviewer subagents after their results are recorded.
- [x] There is a documented and working development-time path to run `harness`
      directly as a command inside this repo without requiring an alias.
- [x] Any behavior-changing implementation in this slice has automated test
      coverage or a clearly justified deterministic validation path.

## Deferred Items

- `harness ui` remains deferred to #2.
- `harness plan list` and docs-navigation follow-ups remain deferred to #4.
- Later skill-system expansion beyond the first dogfoodable pack remains
  deferred to #5.
- Shared test infrastructure remains deferred to #6.

## Work Breakdown

### Step 1: Define the dogfoodable repository entrypoints

- Status: completed

#### Objective

Decide how `harness` should be invoked in development and document that choice
clearly enough that both humans and skills can rely on a stable command name.

#### Details

Do not rely on shell aliases. Prefer a development-time setup path that makes
`harness` directly executable in this repo while staying close to how a future
released binary would be used.

#### Expected Files

- `README.md`
- `AGENTS.md`
- `scripts/install-dev-harness`
- additional repo-local wrapper or setup files only if they are strictly
  needed

#### Validation

- A documented development flow can install or expose a working `harness`
  command in the current environment.
- If setup logic has non-trivial behavior, add or update automated tests where
  practical; otherwise provide deterministic smoke validation steps.

#### Execution Notes

Defined the dev-time entrypoint as `scripts/install-dev-harness` rather than a
shell alias. The installer builds `.local/bin/harness`, links `harness` into a
writable directory on `PATH` when possible, and falls back to `~/.local/bin`
with explicit PATH guidance when no writable PATH entry exists.

#### Review Notes

The first delta review found that installer success could be shadowed by a
different `harness` earlier on `PATH`. The installer now verifies the direct
binary path first, fails if `command -v harness` resolves to a different
binary, and then confirms the repo build is the one the shell will run.

### Step 2: Add human-facing repository docs

- Status: completed

#### Objective

Create the first `README.md` and `AGENTS.md` for `superharness`.

#### Details

`README.md` and `AGENTS.md` should not duplicate each other. README teaches
humans what the project is and how to run it; `AGENTS.md` teaches agents how
to work in the repo and where to find the operational detail.

#### Expected Files

- `README.md`
- `AGENTS.md`

#### Validation

- The README clearly explains the repo purpose, current command surface, and
  development-time setup.
- `AGENTS.md` clearly explains the working agreement, source-of-truth split,
  and lifecycle expectations without embedding every execution detail inline.

#### Execution Notes

Drafted `README.md` and `AGENTS.md` with separate roles: README is for human
onboarding and development setup; `AGENTS.md` defines the repo-level working
agreement, source-of-truth split, lifecycle, and local-skill entrypoints.

#### Review Notes

Fresh-agent dogfood confirmed the README and `AGENTS.md` split is legible.
After the first onboarding pass, the docs were tightened so the controller /
reviewer skill boundary is explicit instead of only inferable from
`harness status`.

### Step 3: Add the first repo-local skill pack

- Status: completed

#### Objective

Add `harness-discovery`, `harness-plan`, `harness-execute`, `harness-land`,
and `harness-reviewer` as the first repo-local skills for `superharness`.

#### Details

Keep the top-level skill surface intentionally small. `harness-execute` should
own the large loop but delegate detail to references.
`harness-reviewer` should be specialized
for reviewer subagents and should not assume the main agent is doing review
submission itself.

#### Expected Files

- `.agents/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-plan/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/*.md`
- `.agents/skills/harness-land/SKILL.md`
- `.agents/skills/harness-reviewer/SKILL.md`

#### Validation

- The skill pack is internally coherent and references `harness --help` and
  `harness <subcommand> --help` instead of duplicating CLI truth unnecessarily.
- `harness-execute` explicitly documents one active review round at a time.
- `harness-execute` explicitly documents that reviewer fan-out must wait for
  all
  reviewer subagents to finish before `harness review aggregate`.
- `harness-execute` or `harness-reviewer` explicitly documents that reviewer
  subagents must be closed after their results are captured to avoid dangling
  background agents.

#### Execution Notes

Drafted the first repo-local skill pack with the five agreed top-level skills:
`harness-discovery`, `harness-plan`, `harness-execute`, `harness-land`, and
`harness-reviewer`. `harness-execute` now points to references for
resume/status, step inner loop, review orchestration,
publish/CI/sync, and closeout/archive. The review orchestration draft includes
the Codex-specific rules to wait for all reviewer subagents, then close them
after their results are consumed.

#### Review Notes

Two reviewer subagents and two pure-context execute testers validated the skill
pack. The first pass surfaced one important installer issue, one minor stale
binary documentation issue, and one discoverability ambiguity around
controller-vs-reviewer responsibilities. A second delta review passed cleanly
after those fixes, and the archive-gate full review (`review-003-full`) also
passed cleanly.

### Step 4: Dogfood the docs and skills against this repository

- Status: completed

#### Objective

Use the new docs and skill contracts to confirm that the repository can steer
its own next loop coherently.

#### Details

The validation should exercise the human-facing entrypoints and the agent-facing
contracts together, not just lint markdown in isolation.

#### Expected Files

- `README.md`
- `AGENTS.md`
- `.agents/skills/**`
- `internal/plan/current.go`
- `internal/plan/current_test.go`
- optional small supporting docs if they improve dogfood legibility

#### Validation

- A fresh agent can identify the intended workflow from `README.md`,
  `AGENTS.md`, and the skill pack without repository-specific hidden context.
- The documented `harness` invocation path works in the current repo.
- Any claimed behavior-changing setup or helper logic is validated by tests or
  deterministic smoke runs.

#### Execution Notes

Dogfooding used both repository-local commands and pure-context subagents:

- ran `go test ./...`
- ran `scripts/install-dev-harness --help`
- ran `scripts/install-dev-harness`
- verified `command -v harness`, `harness --help`, and `harness status`
- started `review-001-delta`, waited for all reviewer subagents, explicitly
  closed them, and aggregated the round
- used a fresh subagent that only read `AGENTS.md`, the execute skill, and
  `harness status` to verify resumability

Dogfooding also surfaced a real status-hand-off bug: an archived
`.local/harness/current-plan.json` could mask a newer active plan. Fixed that in
`internal/plan/current.go` and added regression coverage in
`internal/plan/current_test.go`.

After fixing the review findings, ran `review-002-delta` with the same
wait-all / close-all reviewer flow. That round passed cleanly. A second fresh
execute tester confirmed the controller remains in `execute` during active
review while spawned reviewer subagents use `reviewer`.

#### Review Notes

Dogfood validation passed after the follow-up fixes. The remaining deferred work
is the already-tracked backlog in #2, #4, #5, and #6 rather than new findings
from this slice.

### Step 5: Address revision-2 review feedback for distribution-ready skills

- Status: completed

#### Objective

Absorb the post-PR review feedback that changes the skill pack from a
repo-shaped draft into a more distributable harness skill set.

#### Details

Focus on the structural comments rather than one-off wording nits:

- make discovery more Socratic and option-driven
- make skill naming and descriptions work outside this repository
- tighten the controller-vs-reviewer contract
- move repo-specific bootstrap assumptions out of distributed skills
- make the plan and AGENTS guidance more self-contained

#### Expected Files

- `.agents/skills/**`
- `AGENTS.md`
- `docs/plans/active/2026-03-18-readme-agents-and-skill-pack.md`

#### Validation

- The distributed skill wording no longer assumes the current repository name.
- A fresh agent can tell which harness skill belongs to the controller and
  which belongs to reviewer subagents.
- Discovery guidance now matches the intended Socratic, option-rich
  interaction style.

#### Execution Notes

Renamed the top-level skills to `harness-*` and rewrote the distributed skill
contracts around that naming. `harness-discovery` now follows the missless
style more closely with one high-leverage question per turn, 2-4 realistic
options, and an explicit recommendation. `harness-plan` now stresses
self-contained plan writing and meaningful `YYYY-MM-DD-clear-topic.md`
filenames. `harness-execute`, `harness-land`, and `harness-reviewer` were
tightened so the controller stays in `harness-execute`, reviewer submission
rules live in one file, commit co-author guidance moved into `AGENTS.md`, and
repo-specific bootstrap assumptions moved out of distributed skill wording.

After `review-004-delta`, tightened two controller details:

- `review-orchestration.md` now tells the controller to close a failed reviewer
  agent before respawning that slot
- `resume-and-status.md` now says to follow the repository's documented setup
  path when one exists, and only escalate to the human when no setup path is
  documented
- `harness-reviewer` no longer points subagents at `harness review submit --help`;
  if required input is missing, the reviewer now reports that gap back to the
  controller instead of invoking a second harness command

#### Review Notes

`review-004-delta` found two important controller-contract issues:

- failed reviewer agents were not being closed before respawn
- resume guidance over-indexed on human escalation instead of first using the
  repository's documented setup path

Those findings were fixed locally. `review-005-delta` then found one more
important reviewer-boundary mismatch: the reviewer skill still mentioned
`harness review submit --help` even though the same skill restricted reviewers
to `harness review submit` only. That inconsistency was also fixed locally, and
`review-006-delta` passed cleanly with no blocking or non-blocking findings.

### Step 6: Address revision-3 review feedback on skill wording and metadata

- Status: completed

#### Objective

Absorb the next review pass on the skill pack and archived-plan metadata so the
distributed contracts are simpler, less ambiguous, and better aligned with the
intended lifecycle.

#### Details

Focus on the remaining contract-shape comments:

- move always-needed execute guidance into the main execute skill
- keep discovery focused on plan handoff instead of lifecycle jumps
- clarify review-spec field semantics and remove reviewer-skill duplication
- record the issue-backed provenance and shell/heredoc guidance in the right
  places

#### Expected Files

- `.agents/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`
- `.agents/skills/harness-reviewer/SKILL.md`
- `AGENTS.md`
- `docs/plans/active/2026-03-18-readme-agents-and-skill-pack.md`

#### Validation

- The execute skill includes the resume/start guidance that every controller
  run needs.
- Discovery only hands off into plan writing.
- Review-orchestration explains `kind` and `trigger` clearly and keeps
  reviewer-specific behavior in the reviewer skill.
- Repo metadata reflects the issue-backed source and shell-writing guidance.

#### Execution Notes

Moved the execute resume/status guidance into `harness-execute` itself so the
controller no longer has to jump into a separate reference for every run.
Discovery now reads enough relevant context to ask sharper questions without
pretending "smallest" context is always best, and it now hands off only into
`harness-plan`. Review-orchestration now explains `kind` as a real enum,
`trigger` as a free-form tag with recommended values, and `manifest_path` as a
controller-side artifact rather than a required reviewer input. The fixed
reviewer prompt now passes only the minimal inputs that the reviewer skill
actually needs. `AGENTS.md` also now records the heredoc rule for multi-line
git/gh bodies.

After `review-007-delta`, updated the active plan frontmatter so revision 3 is
clearly issue-backed and review-feedback-driven instead of looking like the
original direct request. After `review-008-delta`, tightened `source_refs` so
the frontmatter stays fully referential and leaves the descriptive revision
story in Step 6 instead of mixing prose into metadata. The controller then ran
`review-009-delta` with a real reviewer subagent plus a fresh cold-start
tester; both confirmed the final provenance cleanup is archive-ready.

#### Review Notes

`review-007-delta` found one important metadata issue: revision 3 still looked
like the original direct request instead of issue-backed follow-up work. That
is now fixed locally by changing the active plan provenance fields.
`review-008-delta` then found one minor follow-up: `source_refs` still mixed a
descriptive label with referential metadata. That non-blocking finding has been
fixed locally. `review-009-delta` then passed with no findings, and a fresh
tester confirmed a cold-start agent could identify the next move from
repository artifacts alone.

## Validation Strategy

- Keep repo-level truth layered:
  README for humans, `AGENTS.md` for repo-level agent norms, and skills for
  execution detail.
- Prefer deterministic command validation over vague prose claims.
- Any new executable setup path for `harness` should be smoke-tested in this
  repo.
- Review the skill pack for clarity, trigger conditions, and hidden-context
  leakage before treating it as dogfood-ready.

## Risks

- Risk: The new docs and skills may reintroduce the same top-level complexity
  that `superharness` is meant to remove from `missless`.
  - Mitigation: Keep the top-level skill set to five entries and push detail
    into `execute` references instead of more peer skills.
- Risk: The repo may document `harness` as a direct command without actually
  making that command reliable in development.
  - Mitigation: Treat the invocation path as part of the deliverable and test
    it explicitly.
- Risk: Reviewer orchestration guidance may stay too generic and fail to encode
  Codex-specific realities around async subagents.
  - Mitigation: Write the wait-for-all and explicit-close rules directly into
    the `execute` and `reviewer` skills.

## Validation Summary

Validated the revision-3 follow-up with deterministic repository checks and
real subagent dogfood:

- `harness status`
- `harness plan lint docs/plans/active/2026-03-18-readme-agents-and-skill-pack.md`
- `git diff --check`
- real reviewer rounds `review-007-delta`, `review-008-delta`, and
  `review-009-delta`
- fresh cold-start tester passes that only used tracked repo artifacts plus
  `harness status`

The final narrow follow-up change is documentation- and contract-only, so the
validation emphasis stayed on lifecycle clarity, provenance, and resumability
instead of additional Go behavior tests.

## Review Summary

Real reviewer subagents and fresh testers drove the revision-3 closeout:

- `review-007-delta` found that revision 3 still looked like the original
  direct request instead of issue-backed follow-up work
- `review-008-delta` found one non-blocking metadata issue: `source_refs` was
  still mixing descriptive text with referential metadata
- `review-009-delta` passed with no findings after the provenance cleanup

The final fresh tester reported no blocking ambiguity in lifecycle, provenance,
or cold-start resumability. Its only nit was that `harness status` does not
yet expose whether every active review slot has already submitted, which is
useful but not required for this slice to archive cleanly.

## Archive Summary

- Archived At: 2026-03-19T23:26:43+08:00
- Revision: 3
- PR: https://github.com/yzhang1918/superharness/pull/10
- Ready: README, `AGENTS.md`, and the first `harness-*` skill pack now agree
  on the repo contract, development install path, controller-vs-reviewer
  boundary, and revision-3 provenance fixes. The final delta review passed
  cleanly and the cold-start tester found no blocking lifecycle ambiguity.
- Merge Handoff: Run `harness archive`, commit and push the tracked plan move,
  then let PR checks rerun before asking for merge approval.

## Outcome Summary

### Delivered

Delivered a dogfoodable repository contract for `superharness`: a human-facing
`README.md`, an agent-facing `AGENTS.md`, a first repo-local `harness-*`
skill pack, and a development-time installer that exposes `harness` directly
as a command in this repo. This revision also tightened the distributed skill
contracts so discovery is more Socratic, execute owns its always-needed resume
guidance, reviewer behavior stays inside the reviewer skill, heredoc guidance
is recorded at the repo level, and the plan metadata is fully issue-backed and
referential.

### Not Delivered

`harness ui`, `harness plan list`, broader skill-system expansion, and shared
test infrastructure remain intentionally deferred from this slice.

### Follow-Up Issues

- #2 `harness ui`
- #4 `harness plan list` and docs navigation
- #5 later skill-system expansion beyond the first dogfoodable pack
- #6 shared test infrastructure
