---
template_version: 0.2.0
created_at: "2026-04-11T22:08:10+08:00"
source_type: direct_request
source_refs: []
size: M
---

# Prepare the v0.2.0 release story and onboarding surface

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare `easyharness` for the `0.2.0` release as the formal public expression
of the existing v0.2 harness contract. The release should present the project
as a real product rather than an internal dogfood alpha, while still stating
clearly that `easyharness` is evolving quickly and may introduce breaking
changes between releases.

The primary user-facing outcome is a product-first README that helps a human
engineer or team understand what `easyharness` is, why harnesses matter,
how to start with `brew install easyharness`, `harness init`, and a coding
agent restart, and how the human role is to steer the work rather than
micromanage the implementation. The supporting outcome is a release-facing
documentation and versioning pass that makes `0.2.0` coherent across the root
`VERSION` file, stable install guidance, and the main user-facing examples.

## Scope

### In Scope

- Bump the repository release version from `0.1.0-alpha.6` to `0.2.0`.
- Reposition the root `README.md` as a mixed product homepage plus concise
  operator entrypoint for teams adopting `easyharness`.
- Keep the README focused on product value, quickstart, steering posture, and
  stable links, while moving detailed local-development setup into a dedicated
  `docs/development.md`.
- Preserve the current fast-development stance: the README should explain that
  breaking changes may happen between releases and that the system is evolving
  toward lower agent cognitive load, stronger execution quality, and better
  human steering without micromanagement.
- Add a small approved supplements package that captures the intended README
  structure and headline copy direction so execution does not depend on chat
  memory.
- Update primary user-facing alpha wording and version examples where they
  would otherwise confuse a new `0.2.0` reader.
- Update focused help text, release guidance, and automated tests only where
  they encode now-stale alpha-only assumptions on the main release path.

### Out of Scope

- Changing the underlying v0.2 workflow, plan schema, or command contract
  semantics beyond the release-facing wording and example updates required by
  this slice.
- Introducing compatibility guarantees, migration shims, or support promises
  that conflict with the repository's fast-development bias.
- Rewriting every historical archived plan or every prerelease-oriented test
  fixture. Historical references and prerelease-specific tests may remain when
  they still serve a real purpose.
- Building a separate marketing site, visual brand system, or screenshot-heavy
  launch asset set outside the repository docs.
- Reworking release automation beyond the minimal changes needed so `0.2.0`
  and future stable releases fit the documented flow.

## Acceptance Criteria

- [x] `VERSION` is `0.2.0`, and the main documented release flow can describe a
      stable `v0.2.0` tag without alpha-only contradictions on primary
      user-facing surfaces.
- [x] `README.md` opens with a product-first explanation that uses the accepted
      positioning, includes a short quickstart centered on `brew install
      easyharness`, `harness init`, and restarting coding agents, and explains
      that humans should steer via plans and execution summaries rather than
      micromanage line-by-line code review.
- [x] `README.md` includes a durable stability statement that says
      `easyharness` is evolving quickly, breaking changes may happen between
      releases, and the burden of tracking internal details should mostly fall
      on the agent following repo-local instructions and state.
- [x] Detailed contributor and local development setup lives in
      `docs/development.md`, with the README pointing there instead of carrying
      a long maintainer-manual section.
- [x] The approved supplements package contains enough README structure and
      copy direction that a future agent can execute the documentation rewrite
      without discovery chat.
- [x] Focused automated tests and release-doc validations cover any changed
      behavior, and prerelease-specific fixtures remain only where they still
      verify real prerelease behavior rather than acting as accidental primary
      examples.

## Deferred Items

- A separate public website, richer launch collateral, or screenshot/video-led
  onboarding beyond repository documentation.
- A long-term compatibility or support policy for future `0.2.x` and `0.3.x`
  releases.
- Broad cleanup of every alpha mention in historical artifacts that are not
  part of the active release-facing surface.

## Work Breakdown

### Step 1: Lock the release positioning and README structure

- Done: [x]

#### Objective

Capture the accepted `0.2.0` release story, README audience, and document
split in tracked form so execution can proceed without hidden context.

#### Details

Discovery already converged on these decisions: the product version should be
`0.2.0`; the release should represent the public expression of the v0.2
contract rather than a compatibility milestone; the README should be a mixed
product homepage rather than a long development manual; and the human role
should be framed as steering, reviewing plans and execution summaries, and
stepping in for judgment rather than micromanaging each line of code. This
step should encode those choices in the tracked plan and in a minimal
supplement draft so later implementation is driven by repository artifacts.

#### Expected Files

- `docs/plans/active/2026-04-11-v0-2-release-story-and-onboarding.md`
- `docs/plans/active/supplements/2026-04-11-v0-2-release-story-and-onboarding/readme-outline.md`

#### Validation

- The tracked plan records the accepted release story, scope, and non-goals.
- The supplements package gives a future agent concrete README structure and
  copy guidance without requiring rediscovery.

#### Execution Notes

Completed during planning by capturing the accepted `0.2.0` release story in
the tracked plan and the matching `readme-outline.md` supplement. That
supplement now records the README hero, quickstart, stability posture, and
human-steering guidance so later implementation does not depend on discovery
chat.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step is the planning artifact itself.

### Step 2: Rewrite the README and split detailed development guidance

- Done: [x]

#### Objective

Turn the root README into a product-first onboarding surface and move the
maintainer-style setup detail into a dedicated development document.

#### Details

The README should lead with the accepted positioning:
`Harnesses matter. Building one shouldn't be the project.` It should then
explain that `easyharness` is a git-native, agent-first harness for
human-steered, agent-executed work; that it makes long-running agent work more
legible and easier to steer; and that humans should review plans, summaries,
and outcomes rather than micromanage every code diff. The quickstart should be
short and practical. The detailed local setup and contributor mechanics
currently living under `Development Setup` should move into a dedicated
document, with README links kept concise and durable. The README may link to
the Anthropic and OpenAI harness essays as background reading when that helps
frame why the product exists.

#### Expected Files

- `README.md`
- `docs/development.md`
- `AGENTS.md` only if a small repo-specific pointer update is needed outside
  the managed block

#### Validation

- The README reads like a product homepage for a human engineer or team
  evaluating adoption, not like a long local setup checklist.
- The quickstart is short, correct, and includes the coding-agent restart
  note after `harness init`.
- Detailed setup instructions are still available through the dedicated
  development doc without weakening the repo's agent-first operating model.

#### Execution Notes

Rewrote `README.md` into a product-first onboarding surface centered on
`Harnesses matter. Building one shouldn't be the project.` The new README now
leads with product positioning, a short Homebrew plus `harness init`
quickstart, a coding-agent restart note, a durable stability statement, and
explicit human-steering guidance. Moved the former long maintainer setup into
`docs/development.md` so the root README no longer acts like the primary local
setup manual. No AGENTS-managed block change was needed for this slice because
the existing root `AGENTS.md` already covers the repo-specific agent operating
entrypoint.

#### Review Notes

Step-closeout full review `review-001-full` requested one important docs fix:
the README release paragraph overstated the Homebrew formula update as
unconditional. Follow-up full review `review-002-full` passed after the README
was corrected to say the tap update only happens when the token is configured.

### Step 3: Align versioning, release guidance, and targeted validation

- Done: [x]

#### Objective

Make the rest of the release-facing surface coherent with `0.2.0` and prevent
primary docs or validations from presenting alpha as the default release line.

#### Details

This step should update the root `VERSION` file plus any user-facing examples,
help snippets, release notes, workflow prompts, or test assumptions that would
mislead a reader about the new stable line. The change should stay disciplined:
keep prerelease-specific behavior and tests where they are still useful, but
do not leave alpha wording on the default install path, release-maintainer
instructions, or the main release guidance. If a script, workflow, or test
uses alpha examples merely as inert sample values, update them when doing so
improves clarity for `0.2.0`; if a test is specifically checking prerelease
semantics, preserve that intent explicitly.

#### Expected Files

- `VERSION`
- `README.md`
- `.github/workflows/release.yml` only if stable example wording needs a
  durable refresh
- `scripts/build-release` and related release helper text only if examples are
  stale on the primary path
- focused files under `tests/smoke/` or `internal/` where alpha-only examples
  have become misleading defaults

#### Validation

- Run focused tests for any changed release/versioning surfaces.
- Verify the planned stable tag resolves correctly from `VERSION`.
- Confirm the main release documentation and scripts no longer imply that the
  default public line is still alpha-only.

#### Execution Notes

Updated the root `VERSION` file to `0.2.0`, refreshed stable-facing sample
versions in `.github/workflows/release.yml`, `scripts/build-release`, and the
CLI version-output test fixture, and rewrote `docs/releasing.md` so the
maintainer path is no longer framed as the project's first alpha release.
Left intentionally prerelease-oriented smoke fixtures alone where they still
exercise prerelease behavior rather than acting as the default public path.
Focused validation so far: `go test ./internal/cli`, `go test ./tests/smoke
-run 'TestRepositoryVersionFileUsesUnprefixedReleaseVersion|TestReadReleaseVersionOutputsVersionAndTag|TestReleaseWorkflowWiresHomebrewTapPublishing'`,
and `scripts/read-release-version --tag` confirming `VERSION=0.2.0` maps to
`v0.2.0`. TDD was not relevant here because the slice changed docs, examples,
and release-facing copy rather than introducing new runtime behavior.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 is a narrow release-facing copy and example
alignment slice tightly coupled to the integrated candidate. Focused validation
already covered the changed version and workflow surfaces, and the full
finalize review will inspect the combined result more meaningfully than a
separate isolated reviewer round here.

## Validation Strategy

- Run `harness plan lint docs/plans/active/2026-04-11-v0-2-release-story-and-onboarding.md`.
- During execution, use focused doc review plus targeted release/version tests
  rather than a broad unrelated test sweep.
- Re-read the final README cold, as if arriving from GitHub without chat
  history, to confirm the product story and quickstart still make sense.

## Risks

- Risk: The README rewrite becomes generic marketing copy and loses the sharp
  explanation of how `easyharness` changes the human and agent workflow.
  - Mitigation: keep the accepted positioning anchored in concrete workflow
    value, quickstart commands, and the `steer, don't micromanage` operating
    model.
- Risk: Moving development setup out of README drops contributor-critical
  information that agents or humans still need.
  - Mitigation: move the detail into `docs/development.md`, keep README links
    explicit, and update `AGENTS.md` only if a small pointer is genuinely
    necessary.
- Risk: The versioning cleanup accidentally deletes prerelease-specific tests
  or wording that still verifies a real prerelease path.
  - Mitigation: distinguish primary release-surface examples from intentional
    prerelease fixtures and preserve the latter when they still test behavior.

## Validation Summary

- `harness plan lint docs/plans/active/2026-04-11-v0-2-release-story-and-onboarding.md`
- `go test ./internal/cli`
- `go test ./tests/smoke -run 'TestRepositoryVersionFileUsesUnprefixedReleaseVersion|TestReadReleaseVersionOutputsVersionAndTag|TestReleaseWorkflowWiresHomebrewTapPublishing'`
- `go test ./tests/smoke -run 'TestBuildReleaseProducesStableArchiveAndVersionedBinary|TestBuildReleaseHelpUsesStableExampleVersion|TestReleaseWorkflowWiresHomebrewTapPublishing|TestRepositoryVersionFileUsesUnprefixedReleaseVersion|TestReadReleaseVersionOutputsVersionAndTag'`
- `go test ./tests/smoke -run 'TestReleaseDocsPresentStableOnboardingSurface|TestBuildReleaseProducesStableArchiveAndVersionedBinary|TestBuildReleaseHelpUsesStableExampleVersion|TestReleaseWorkflowWiresHomebrewTapPublishing|TestRepositoryVersionFileUsesUnprefixedReleaseVersion|TestReadReleaseVersionOutputsVersionAndTag'`
- `scripts/read-release-version --tag`

## Review Summary

- Step-closeout full review `review-001-full` on Step 2 requested one blocking
  docs fix because the new README implied every release updates the Homebrew
  tap formula. The repair clarified that release assets are always published,
  while the tap update remains conditional on the configured token.
- Follow-up full review `review-002-full` passed for Step 2 with zero blocking
  and zero non-blocking findings.
- Finalize full review `review-003-full` requested two blocking repairs:
  `docs/development.md` still contained workstation-specific paths, and the
  stable `v0.2.0` release-build path still lacked dedicated smoke coverage.
- Finalize full review `review-004-full` passed after those repairs, with zero
  blocking and zero non-blocking findings.
- Revision 2 reopened the archived candidate in `finalize-fix` mode because
  sync evidence showed the branch was stale against `origin/main`. The repair
  merged the latest `origin/main` cleanly, reran focused validation, and then
  resumed finalize review.
- Finalize full review `review-005-full` requested one blocking repair because
  the rewritten docs surfaces still lacked a focused regression guard. That
  led to the new docs smoke test.
- Finalize full review `review-006-full` requested one blocking docs fix
  because `docs/releasing.md` understated the fact that prerelease tags also
  update the Homebrew tap when the token is configured.
- Finalize full review `review-007-full` passed after those repairs, with one
  non-blocking tests finding noting that the docs smoke test could later pin
  the token-gated prerelease clause even more tightly if the repository wants
  a stricter future guard.

## Archive Summary

- Archived At: 2026-04-11T22:50:35+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/143
- Ready: The candidate now presents `easyharness` as a product-first `0.2.0`
  release, keeps the rapid-iteration and steer-not-micromanage posture clear,
  moves detailed maintainer setup into `docs/development.md`, updates stable
  release-facing examples, includes focused smoke coverage for the rewritten
  docs surfaces, absorbs the latest `origin/main` cleanly in revision 2, and
  passed finalize full review `review-007-full`.
- Merge Handoff: Revision 2 is archived. Push the updated branch to refresh PR
  #143, record fresh publish/CI/sync evidence for revision 2, and wait for
  merge approval once status reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Rewrote `README.md` into a product-first onboarding surface centered on
  `Harnesses matter. Building one shouldn't be the project.`
- Added a short quickstart built around Homebrew install, `harness init`, and
  restarting the coding agent so repo-local instructions and skills take
  effect cleanly.
- Added a durable stability statement that keeps the rapid-iteration and
  breaking-change posture explicit while telling the human operator that the
  agent should recover workflow detail from repo-local state and instructions.
- Framed the human role as steering through plans, summaries, and high-signal
  artifacts rather than micromanaging every implementation detail.
- Moved detailed local development and maintainer setup into
  `docs/development.md`, keeping the root README shorter and more product-like.
- Updated `VERSION`, release docs, workflow/help examples, and the CLI
  version-output fixture to align the primary release path with `0.2.0`.
- Fixed finalize-review feedback by removing workstation-specific development
  doc paths and adding focused stable release-build smoke coverage plus stable
  example-string assertions.
- Reopened the archived candidate in revision 2 to absorb the latest
  `origin/main` cleanly, then added focused smoke coverage for the rewritten
  README, development doc, and release guide plus the clarified prerelease tap
  wording in `docs/releasing.md`.

### Not Delivered

- No separate marketing site, screenshot-heavy launch collateral, or richer
  visual launch package was added in this slice.
- No long-term compatibility or support policy for future `0.2.x` or `0.3.x`
  releases was introduced in this slice.
- No broad cleanup of every alpha mention in historical archived artifacts was
  attempted beyond the active release-facing surfaces.

### Follow-Up Issues

- No new follow-up issue was created in this slice. The deferred items remain
  future packaging and policy work such as a separate public website or a
  longer-term compatibility story if the project later wants to formalize
  those tradeoffs. One non-blocking review note also remains available for
  future tightening: the docs smoke test could pin the token-gated prerelease
  clause even more explicitly if the team later wants an even stricter guard.
