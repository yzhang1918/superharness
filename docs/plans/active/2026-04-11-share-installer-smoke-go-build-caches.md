---
template_version: 0.2.0
created_at: 2026-04-11T10:56:23+08:00
source_type: direct_request
source_refs: ["chat://current-session"]
size: XS
---

# Share Installer Smoke Go Build Caches

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Reduce the wall-clock cost of the installer-focused smoke suite without
changing installer behavior or broadening cache-sharing policy across the rest
of `tests/smoke`.

Discovery showed that `go test ./tests/smoke -count=1` currently takes about
223 seconds locally, with most of the time concentrated in
`tests/smoke/install_dev_harness_test.go`. Those installer cases repeatedly
invoke `scripts/install-dev-harness`, and the helper currently gives each test
its own `GOCACHE` and `GOMODCACHE`, forcing repeated cold `go build` work. The
intended outcome is to keep installer test isolation for `HOME`, fixture
repositories, install paths, and workdirs while sharing only the Go build and
module caches used by that installer smoke group.

## Scope

### In Scope

- Update installer smoke helpers so `install_dev_harness_test.go` reuses a
  package-level `GOCACHE` and `GOMODCACHE`.
- Keep the cache-sharing policy narrowly scoped to the installer smoke suite.
- Preserve the current installer behavior assertions and test coverage surface.
- Re-run installer-focused smoke coverage and the full smoke package to confirm
  the optimization does not change outcomes.

### Out of Scope

- Changing installer runtime behavior, wrapper dispatch rules, or fallback
  semantics.
- Expanding shared Go caches to all of `tests/smoke` or other test packages.
- Reclassifying, removing, or downgrading installer smoke coverage.
- CI workflow shape changes beyond whatever runtime improvement falls out of
  the faster smoke package.

## Acceptance Criteria

- [ ] Installer smoke helpers provide a shared `GOCACHE` and `GOMODCACHE`
      across `install_dev_harness_test.go` while keeping per-test fixture and
      environment isolation for non-cache state.
- [ ] The cache-sharing policy remains local to installer smoke support and
      does not silently change execution environments for unrelated smoke
      tests.
- [ ] Installer-focused smoke coverage still passes with the new helper model.
- [ ] A fresh `go test ./tests/smoke -count=1` run shows a meaningful runtime
      reduction relative to the discovery baseline, with the before/after
      evidence recorded in execution notes before archive.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Narrow installer smoke cache reuse to build artifacts only

- Done: [x]

#### Objective

Refactor installer smoke support so repeated installer invocations reuse Go
build and module caches without weakening the existing per-test functional
isolation.

#### Details

The current bottleneck lives in `installerEnv`, which assigns a fresh temp
`GOCACHE` and `GOMODCACHE` for every test even though the expensive work is the
same repeated `go build` inside `scripts/install-dev-harness`. Introduce
package-level shared cache directories for this installer smoke file only, then
keep per-test values for `HOME`, temp repos, PATH fixtures, install dirs, and
other stateful paths. Avoid broad helper changes that would implicitly alter
the environment model for unrelated smoke suites.

#### Expected Files

- `tests/smoke/install_dev_harness_test.go`

#### Validation

- `go test ./tests/smoke -run TestInstallDevHarness -count=1`
- Spot-check that the changed helper still leaves each test on independent
  temp directories for non-cache state.

#### Execution Notes

Updated `tests/smoke/install_dev_harness_test.go` so installer smoke defaults
to package-level shared `GOCACHE` and `GOMODCACHE` directories created once per
test process, while preserving per-test `HOME`, temp fixture repositories,
install directories, and other stateful paths. The helper still respects
explicit cache overrides when a caller provides them. Focused validation passed
with `/usr/bin/time -p go test ./tests/smoke -run TestInstallDevHarness -count=1`,
which completed in about `42.5s` after the helper change.

#### Review Notes

NO_STEP_REVIEW_NEEDED: XS test-only helper change with no installer behavior
contract change; a broader finalize review will cover the branch candidate.

### Step 2: Prove the optimization stays narrow and worthwhile

- Done: [x]

#### Objective

Validate that the cache reuse improves smoke runtime while leaving the rest of
the smoke package and installer assertions unchanged.

#### Details

Capture before/after timing evidence from the installer subset or the full
smoke package so archive-time readers can tell whether the change delivered a
real speedup. Re-run the full smoke package after the helper change to ensure
the narrower cache-sharing policy did not accidentally leak into unrelated
tests. If the measured gain is marginal or exposes hidden coupling, stop and
record that outcome instead of stretching the slice into broader test
reorganization.

#### Expected Files

- `tests/smoke/install_dev_harness_test.go`
- `docs/plans/active/2026-04-11-share-installer-smoke-go-build-caches.md`

#### Validation

- `go test ./tests/smoke -count=1`
- Record comparative timing evidence in the plan execution notes before
  archive.

#### Execution Notes

Measured the full smoke package after the helper change with
`/usr/bin/time -p go test ./tests/smoke -count=1`, which completed in about
`70.0s` (`ok ... 69.795s`, `real 69.96`). Discovery baseline before the change
was about `222.6s`, so the package-level runtime dropped by roughly `152.6s`
without broadening shared-cache policy outside installer smoke support.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only captured timing evidence and confirmed
the scope boundary; finalize review is a better fit than a separate step review
for this tiny branch candidate.

## Validation Strategy

- Run the installer-focused smoke subset first so the helper change is checked
  against the concentrated risk surface.
- Re-run the full `tests/smoke` package to verify unchanged behavior and to
  measure the end-to-end payoff against the discovery baseline of about 223
  seconds.
- If timing varies materially between runs, report the range and the command
  context instead of overstating certainty.

## Risks

- Risk: Shared Go caches could accidentally mask a real installer failure that
  only appears with a cold cache.
  - Mitigation: Restrict reuse to installer smoke only, keep all non-cache
    state isolated per test, and preserve a full smoke rerun after the helper
    change.
- Risk: A helper refactor intended for installer smoke could silently alter
  unrelated smoke environments.
  - Mitigation: Keep the implementation local to
    `tests/smoke/install_dev_harness_test.go` unless a narrower shared helper
    proves unavoidable, and treat any broader surface change as out of scope
    for this slice.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
