# Testing Structure Proposal

## Status

This document is a non-normative proposal.

It describes a recommended testing structure for `superharness`. It does not
change the current normative CLI or plan contracts by itself.

## Purpose

`superharness` already has a strong package-level Go test suite for plan,
status, review, lifecycle, and CLI behavior. The repository does not yet have
a durable structure for top-level smoke, end-to-end, or resilience testing.

This proposal defines how those suites should be organized without turning the
repository into a scripts-heavy test harness.

## Goals

- keep unit and package-level contract tests close to the code they exercise
- add a clear home for repo-level smoke, end-to-end, and resilience tests
- keep the real `harness` binary as the system under test for higher-level
  suites
- standardize on one automation entrypoint for automated tests
- minimize duplicated fixtures and full-repository snapshots
- keep tests understandable to both humans and agents

## Non-Goals

- redefining the current CLI output contract
- redefining the tracked plan schema
- introducing external service dependencies for test orchestration
- requiring large checked-in repository snapshots for every scenario
- creating a separate top-level `ft/` taxonomy unless future scope makes it
  clearly necessary

## Design Principles

### Keep Lower-Level Tests Near the Code

Pure logic tests and package-level contract tests should stay next to the code
under `internal/*/*_test.go`.

This includes:

- unit tests for narrow helpers and parsing rules
- package-level contract tests that exercise several functions inside one
  package
- in-process CLI tests that call Go entrypoints directly instead of launching
  an external process

The current repository already follows this model well.

### Use `go test` as the Test Runner

Repo-level smoke, end-to-end, and resilience suites should still be invoked by
`go test`, even when the test itself launches the real `harness` binary.

The distinction is:

- `go test` is the test runner
- the built `harness` binary is the system under test

This keeps:

- assertions precise and readable
- temporary workspace creation simple
- test filtering and reruns consistent
- CI integration straightforward
- the repository aligned with Go-native tooling

Optional shell helpers may exist, but they should remain thin wrappers around
`go test` rather than becoming a second test framework.

### Test the Built Binary, Not the PATH

Repo-level higher-level suites should build `./cmd/harness` into a temporary
path and execute that binary directly.

They should not rely on whichever `harness` command currently appears on the
developer's `PATH`, because that binary may not match the working tree under
test.

### Prefer Minimal Generated Workspaces Over Large Snapshots

Tests should build the smallest possible temporary workspace for the scenario,
then mutate it as needed.

Checked-in fixtures should be reserved for stable, hard-to-generate, or
historical compatibility cases such as:

- corrupted local state payloads
- legacy artifact layouts
- intentionally malformed tracked files
- compact reusable workspace seeds

Avoid storing many full repository copies under `tests/testdata/` unless the
repository later proves that such snapshots are materially easier to maintain
than generated workspaces.

## Proposed Taxonomy

### Unit and Package-Level Contract Tests

Location:

- `internal/*/*_test.go`

Purpose:

- validate narrow logic and package-owned contract behavior

Execution model:

- in-process
- may use temporary files or directories
- should remain fast and be part of the default `go test ./...` path

Examples in the current repository include plan linting, plan parsing, status
state inference, review round artifact logic, and archive/reopen behavior.

### Smoke Tests

Location:

- `tests/smoke/`

Purpose:

- provide a fast confidence check that the binary starts and the most critical
  user-visible paths are not obviously broken

Characteristics:

- few cases
- fast runtime
- real binary execution
- shallow assertions compared with end-to-end tests

Typical smoke coverage for `superharness` should include:

- `harness --help`
- `harness status`
- `harness plan template`
- a minimal `plan template -> plan lint` roundtrip

### End-to-End Tests

Location:

- `tests/e2e/`

Purpose:

- exercise real user flows through the built binary against a temporary
  workspace that looks like an actual harness repository

Characteristics:

- real process execution
- multiple commands per scenario
- assertions on command outputs, tracked files, and local artifacts

Typical `superharness` E2E scenarios should include:

- happy-path plan creation and lint
- review-round start, submit, and aggregate flow
- archive and reopen lifecycle roundtrip
- landed-state reporting after `harness land --pr ...` and `harness land complete`

### Resilience Tests

Location:

- `tests/resilience/`

Purpose:

- verify that the system fails safely and predictably when local state,
  tracked files, or filesystem operations are incomplete, stale, or malformed

Why `resilience` instead of `chaos`:

- for a local CLI and repository contract, most high-value fault cases are
  deterministic failure-injection scenarios rather than distributed-system
  chaos experiments

Typical `superharness` resilience coverage should include:

- corrupted `.local/harness/current-plan.json`
- missing or unreadable review aggregate artifacts
- archive operations that fail mid-write and must roll back cleanly
- conflicting active plans or ambiguous current-plan pointers
- stale sync or CI state that must block archive or merge-readiness claims

## Proposed Directory Layout

```text
docs/
internal/
tests/
  support/
    binary.go
    repo.go
    run.go
    assert.go
  testdata/
    minimal-repo/
    corrupted-state/
    legacy-review-round/
  smoke/
    smoke_test.go
  e2e/
    happy_path_test.go
    review_round_test.go
    archive_reopen_test.go
  resilience/
    corrupted_state_test.go
    archive_rollback_test.go
    ambiguous_current_plan_test.go
```

### `tests/support/`

Shared helpers for:

- building the test binary once per suite or package
- creating temporary harness workspaces
- running commands and collecting stdout, stderr, and exit codes
- parsing JSON envelopes and common artifacts

This directory should keep repo-level tests concise and avoid repeated shell
wrappers.

### `tests/testdata/`

Shared checked-in fixtures for:

- reusable minimal workspace seeds
- malformed JSON or tracked markdown inputs
- legacy artifact compatibility cases

Tests should not depend on large or numerous full repository snapshots by
default.

## Fixture Strategy

The repository should prefer this order:

1. generate the scenario inside the test
2. start from a small checked-in seed and mutate it
3. use a fully checked-in scenario only when generation would be materially
   harder to understand or maintain

In practice, this means:

- create temporary repositories in code for happy-path flows
- keep a few durable fixtures for malformed or legacy inputs
- avoid duplicating near-identical workspace trees across many test cases

## Naming Guidance

This proposal intentionally does not add a top-level `tests/ft/` or
`tests/integration/` directory.

Reasons:

- current package-local Go tests already cover most useful integration points
- `functional` is ambiguous for a CLI-first repository
- `end-to-end` is a clearer label for real binary-driven workflow tests
- `resilience` is a clearer label than `chaos` for the repository's current
  failure modes

If future scope introduces external services or a UI, the taxonomy may expand.

## Execution Model

Recommended commands:

- `go test ./...`
  - default package-local suite
- `go test ./tests/smoke -count=1`
  - fast repo-level smoke coverage
- `go test ./tests/e2e -count=1`
  - real binary workflow coverage
- `go test ./tests/resilience -count=1`
  - deterministic failure-injection coverage

Optional wrapper scripts may exist, for example:

- `scripts/test-smoke`
- `scripts/test-e2e`

Those wrappers should delegate to the Go-based suites instead of embedding a
second assertion layer in shell.

## Adoption Plan

The repository should adopt this proposal incrementally.

### Phase 1

- add `tests/support/`
- add `tests/smoke/`
- add a small smoke suite that builds and runs the real binary

### Phase 2

- add one happy-path `tests/e2e/` scenario
- cover the main workflow from plan creation through archive or land status

### Phase 3

- add a focused `tests/resilience/` suite
- begin with a small number of deterministic failure cases that are already
  known to matter for archive and status correctness

## Initial Suggested Scenarios

The first repo-level cases should be:

- smoke: `harness --help`
- smoke: `harness status` in a temporary minimal workspace
- smoke: `plan template -> plan lint`
- E2E: `plan template -> review start -> review submit -> review aggregate`
- E2E: `archive -> status -> reopen -> status`
- resilience: corrupted current-plan marker
- resilience: archive rollback after a write failure

## Open Questions

- whether `go test ./...` should eventually include `tests/smoke` by default
  or keep repo-level suites opt-in
- whether any repo-level suite should use build tags such as `resilience`
  instead of dedicated package paths
- whether future release packaging should add a separate release-verification
  smoke path distinct from repository development smoke coverage
