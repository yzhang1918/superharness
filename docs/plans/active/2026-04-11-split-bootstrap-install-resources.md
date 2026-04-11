---
template_version: 0.2.0
created_at: "2026-04-11T15:14:01+08:00"
source_type: direct_request
source_refs:
    - chat://current-session
size: M
---

# Split Bootstrap Install Into Init, Skills, and Instructions Resources

## Goal

Replace the current monolithic `harness install` bootstrap flow with a clearer
resource model that matches user intent: an idempotent `harness init`
entrypoint for bootstrap and refresh, plus noun-first resource commands for
managing repo/user skills and instructions independently.

This slice should leave easyharness with a cleaner end-state rather than a
compatibility bridge. The new flow should define explicit scope semantics,
versioned managed assets, and safe ownership rules when a repository or user
already has custom skills or instruction files in place.

## Scope

### In Scope

- Introduce `harness init` as the default quick-start bootstrap and refresh
  entrypoint for the current repository.
- Replace the existing umbrella `harness install` interface with resource
  commands shaped around `harness skills ...` and `harness instructions ...`.
- Define and implement explicit scope semantics for repo-local and user-level
  installs, using one stable vocabulary across CLI help, docs, and tests.
- Support agent-aware default targets while preserving explicit override hooks
  such as `--dir` for skills and `--file` for instructions.
- Add a dedicated bootstrap install spec that defines the resource model,
  supported agent adapter surface, ownership/version rules, and current support
  matrix so CLI contracts and docs can reference one normative source.
- Install managed bootstrap skills in a standards-aligned skill package layout,
  including version metadata tied to the easyharness release version.
- Add version information to the managed instructions block so installed
  bootstrap content can be identified and refreshed deterministically.
- Define safe ownership and conflict rules so easyharness-managed assets do not
  silently overwrite unrelated user-owned skills or instruction files.
- Update bootstrap asset sources, docs, and tests to match the new command and
  ownership contract.

### Out of Scope

- Full first-class Claude Code bootstrap profiles beyond supporting explicit
  alternate `--file` and `--dir` targets plus any minimal agent-selection
  plumbing needed to keep the interface extensible.
- Backward-compatibility shims, fallback command aliases, or dual-write logic
  for the old `harness install` surface unless a blocker discovered during
  execution makes a narrow exception necessary.
- Non-bootstrap skill lifecycle features such as remote discovery, search,
  marketplace installation, or arbitrary third-party skill updates.
- A separate global state database or hidden install registry outside the
  repository or skill packages themselves.

## Acceptance Criteria

- [x] `harness init` bootstraps the current repository using the default agent
      profile, is safe to rerun idempotently, and can refresh managed assets
      after an easyharness version upgrade.
- [x] Resource-level commands exist for skills and instructions, with noun-first
      shapes such as `harness skills install|uninstall` and
      `harness instructions install|uninstall`.
- [x] Scope semantics are explicit and consistent for repository and user
      targets, with agent-specific defaults plus `--dir` and `--file`
      overrides for non-default layouts.
- [x] A dedicated bootstrap install spec exists and is the primary normative
      reference for resource semantics, support boundaries, ownership/version
      rules, and external skill-format references.
- [x] Managed bootstrap skills install as valid skill packages and carry
      easyharness release version metadata in standard `SKILL.md` frontmatter
      rather than private hidden manifest files.
- [x] The managed instructions block carries an in-band easyharness version so
      refresh behavior can distinguish current content from stale content.
- [x] Installing or uninstalling managed skills/instructions never silently
      overwrites unrelated user-owned assets; collisions surface a clear error
      unless the target is already recognized as easyharness-managed.
- [x] Tests and docs cover the Codex default targets plus explicit alternate
      path overrides that make the bootstrap flow usable for other agent
      ecosystems before full native profiles ship.

## Deferred Items

- Native multi-agent profile packs with polished defaults for Claude Code or
  other non-Codex agents.
- Marketplace or registry-driven global skill installation beyond the bootstrap
  pack shipped with easyharness itself.
- Rich inspection commands such as `harness skills doctor` or
  `harness instructions show` unless execution shows they are required to make
  the new contract understandable.

## Work Breakdown

### Step 1: Define the new bootstrap command and ownership contract

- Done: [x]

#### Objective

Revise the CLI contract and bootstrap asset model so `init`, `skills`, and
`instructions` have clear responsibilities, explicit scope semantics, and a
durable ownership/version strategy.

#### Details

Document the clean target design before changing code: `harness init` should be
the default repo bootstrap entrypoint and rerunnable refresh path, while
granular lifecycle actions move to resource commands. This step should settle
the preferred scope vocabulary, expected defaults for agent-aware targets,
collision behavior when assets already exist, refresh behavior after version
upgrades, and where version/ownership information lives. Create a dedicated
bootstrap install spec so CLI contracts, README guidance, and future agent
adapters can reference one normative description. That spec should cite the
Agent Skills format specification at
`https://agentskills.io/specification.md` as the baseline skill layout contract and
the Codex skills guide at
`https://developers.openai.com/api/docs/guides/tools-skills.md` for Codex-
specific metadata/extensions. For managed skills, use standards-aligned
`SKILL.md` frontmatter metadata keyed to the easyharness release version rather
than inventing hidden per-skill manifests. For managed instructions, add an
in-band version marker inside the managed block so reruns can reason about
installed content without rewriting unrelated user-owned prose.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/bootstrap-install.md`
- `README.md`
- `assets/bootstrap/agents-managed-block.md`
- `assets/bootstrap/skills/**`

#### Validation

- The updated CLI contract describes `init`, `skills`, and `instructions`
  without leaving behavior dependent on chat context.
- The new bootstrap install spec is self-contained enough that CLI contracts
  and future agent adapters can reference it instead of restating the same
  rules.
- The documented ownership/version model clearly distinguishes managed assets
  from unrelated user-owned assets.
- The chosen design remains compatible with the published skill spec and Codex
  optional metadata conventions.

#### Execution Notes

Defined the clean bootstrap target model around `harness init`,
`harness skills install|uninstall`, and
`harness instructions install|uninstall`. Added a dedicated normative spec at
`docs/specs/bootstrap-install.md`, updated the CLI contract and spec index to
reference it, clarified that `init` is idempotent and rerunnable after version
upgrades, and documented standards-aligned skill metadata plus versioned
managed-block markers.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The command model, bootstrap spec, and user-facing docs
were implemented as one tightly coupled slice and are better covered by the
candidate-level finalize review than by an artificial early review boundary.

### Step 2: Implement resource commands and managed-asset safety rules

- Done: [x]

#### Objective

Replace the existing install implementation with the new command surface and
enforce safe create/update/remove behavior for versioned managed skills and
instructions.

#### Details

This step should reshape the CLI entrypoints and install service code around
resource-specific actions instead of the current dual-purpose `install`
command. Repo and user scopes should both be supported where they make sense,
with agent-aware defaults for Codex and explicit override flags for alternate
layouts. Managed skills should be installable and uninstallable based on
standards-aligned frontmatter metadata plus easyharness version matching, while
user-owned skill packages remain untouched. Instructions management should
update only the managed block or target file and preserve non-managed content.
`harness init` should call into the same underlying resource install logic so
bootstrap and refresh semantics stay consistent. Because the repository is in
fast-development mode, prefer the clean command cutover over compatibility
wrappers for the old shape.

#### Expected Files

- `internal/cli/app.go`
- `internal/install/service.go`
- `internal/contracts/*.go`
- `schema/commands/*.json`
- `cmd/harness/main.go`

#### Validation

- Targeted unit tests cover create, update, conflict, and uninstall paths for
  both skills and instructions.
- Dry-run behavior reports the intended actions for the new command surface.
- Existing repo-local user content survives reruns unless it is explicitly
  recognized as easyharness-managed.

#### Execution Notes

Replaced the old single `harness install` command surface with resource-aware
CLI entrypoints for `init`, `skills`, and `instructions`. Refactored the
bootstrap install engine to support repo/user scopes, codex defaults plus
explicit overrides, versioned managed-block rendering, standards-aligned skill
frontmatter metadata, safe uninstall flows, stale managed-skill cleanup, and
conflict errors when same-path skills are not recognized as easyharness-
managed. Preserved refresh behavior for legacy installs by recognizing older
packaged skills that predate the new metadata markers.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The parser, service, and ownership behavior changed as
one integrated contract rewrite; finalize review is the meaningful review
boundary for this slice.

### Step 3: Refresh packaged bootstrap assets, docs, and end-to-end coverage

- Done: [x]

#### Objective

Bring packaged assets, repository dogfood outputs, and test coverage into sync
with the new bootstrap command model and managed version metadata.

#### Details

After the command and ownership behavior land, refresh the packaged bootstrap
instructions block and skill pack so the shipped assets match the new contract.
Update repo docs to present `harness init` as the quick-start path and explain
when to use `harness skills ...` or `harness instructions ...` for finer
control. Coverage should prove the new repo bootstrap path, rerunnable refresh
behavior after version changes, alternate path overrides, and safety behavior
around pre-existing user-owned assets. If bootstrap source files change, rerun
the repo sync flow so `.agents/skills/` and the root managed `AGENTS.md` block
stay aligned with `assets/bootstrap/`.

#### Expected Files

- `tests/smoke/**`
- `internal/install/*_test.go`
- `scripts/sync-bootstrap-assets`
- `.agents/skills/**`
- `AGENTS.md`

#### Validation

- `scripts/sync-bootstrap-assets --check`
- Relevant Go test suites covering CLI, install logic, and smoke bootstrap
  behavior
- Manual dry-run spot checks for repo and user target flows if automated
  coverage does not fully exercise both paths

#### Execution Notes

Updated repo-level docs and dogfood materialized outputs, regenerated contract
schemas, and refreshed `.agents/skills/` plus the root managed `AGENTS.md`
block through the sync scripts so the repository reflects the new bootstrap
contract. Added or updated package, CLI, bootstrapsync, and smoke coverage for
the new entrypoints, version markers, safety rules, and rerunnable refresh
behavior.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This validation and dogfood follow-through depends on
the Step 1/2 contract changes and is most usefully reviewed at the full
candidate level.

## Validation Strategy

- Use unit tests to pin command parsing, ownership/version detection, and
  install/uninstall safety behavior.
- Use smoke coverage to validate end-to-end bootstrap flows for repo defaults
  and explicit alternate paths.
- Re-run `scripts/sync-bootstrap-assets` and its drift check whenever packaged
  bootstrap assets change.
- Re-run any generated schemas or reference docs required by the updated CLI
  contract.

## Risks

- Risk: The new command split could make the first-run experience more complex
  even if the underlying model is cleaner.
  - Mitigation: Keep `harness init` as the documented default path and reserve
    resource commands for finer control.
- Risk: Refresh semantics could be ambiguous if `harness init` behaves
  differently from the underlying resource install commands after a version
  upgrade.
  - Mitigation: Route `init` through the same resource install engine and add
    coverage for rerun/update behavior when the easyharness version changes.
- Risk: Ownership detection could still overwrite user assets if the metadata
  or managed-block version rules are too loose.
  - Mitigation: Require explicit recognizable managed markers/version metadata
    before updating or uninstalling existing assets, and add tests for
    same-path collisions.
- Risk: Supporting agent-aware paths too early could overfit to placeholder
  abstractions without delivering useful alternate-agent behavior.
  - Mitigation: Keep the first slice narrow: Codex defaults first, explicit
    `--dir`/`--file` overrides second, richer agent-native profiles deferred.

## Validation Summary

- `go test ./internal/install ./internal/bootstrapsync ./internal/cli -count=1`
  passed after the ownership-safety follow-up.
- `go test ./tests/smoke -run 'TestHelpShowsTopLevelUsage|TestInit|TestSkills|TestInstructions' -count=1`
  passed after adding user-scope and explicit-override smoke coverage.
- `scripts/sync-bootstrap-assets --check` and
  `scripts/sync-contract-artifacts --check` both passed after the final
  contract and asset refresh.
- Additional spot checks during repair work confirmed the new repo bootstrap,
  version-refresh, user-scope, and explicit-target flows without relying on
  backward-compatibility shims.

## Review Summary

- `review-001-full` found 6 blocking findings across ownership safety and test
  coverage; those drove the follow-up repair work.
- `review-002-delta` passed after the ownership and service-test fixes, with 1
  non-blocking finding about missing binary-level smoke coverage for user-scope
  and explicit override paths.
- `review-003-delta` passed after adding binary-level smoke coverage for
  `harness init` explicit overrides plus user-scope resource installs.

## Archive Summary

PR: Branch-local candidate is ready for publish handoff; no PR metadata was
recorded during this local execution loop.

Ready: The candidate reached a passing finalize review state after one full
review and two narrow delta follow-ups, with no blocking findings remaining.

Merge Handoff: After archive, refresh publish/CI/sync evidence and then wait
for human merge approval once the archived candidate has current publish, CI,
and sync evidence recorded.

## Outcome Summary

### Delivered

- Replaced the monolithic bootstrap install flow with `harness init`,
  `harness skills install|uninstall`, and
  `harness instructions install|uninstall`.
- Added a dedicated bootstrap install spec and aligned README, CLI contract,
  bootstrap assets, and dogfood outputs with the new resource model.
- Switched managed skills to standards-aligned package metadata in `SKILL.md`
  and added in-band easyharness version markers to the managed instructions
  block.
- Implemented safer ownership detection so easyharness-managed assets refresh
  cleanly while same-name user-owned skills or instructions fail closed.
- Added targeted service, bootstrapsync, CLI, and smoke coverage for idempotent
  reruns, version refreshes, user scope, and explicit override targets.

### Not Delivered

None within the approved scope.

### Follow-Up Issues

- Deferred roadmap items remain intentionally out of scope for this slice:
  native multi-agent profile packs beyond explicit override hooks,
  marketplace-style skill distribution, and richer inspection commands such as
  `harness skills doctor` or `harness instructions show`.
