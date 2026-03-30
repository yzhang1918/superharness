# Releasing

`easyharness` ships its first public alpha as GitHub Release archives built
from the tracked release workflow at
[catu-ai/easyharness](https://github.com/catu-ai/easyharness).

The release archive name follows the project name, while the unpacked
executable remains `harness`. Tagged releases can also update the dedicated
Homebrew tap formula `easyharness` in `catu-ai/homebrew-tap`, which users
install as `catu-ai/tap/easyharness`.

## Release Checklist

1. Decide the next version tag, such as `v0.1.0-alpha.1`.
2. Make sure `main` is up to date and the release branch is clean.
3. Run `go test ./...`.
4. Run `scripts/build-release --version <tag>` locally if you want to verify the
   packaging path before publishing.
5. Create and push the matching git tag, for example
   `git tag v0.1.0-alpha.1 && git push origin v0.1.0-alpha.1`.
6. Confirm the release workflow uploaded the release archives and
   `SHA256SUMS` file.
7. If the Homebrew tap token is configured, confirm the workflow updated
   `Formula/easyharness.rb` in `catu-ai/homebrew-tap`.
8. Confirm the release workflow's Homebrew verification job passed.
   When the tap token is configured, it should install `catu-ai/tap/easyharness`
   and pass `brew test easyharness`. Without the token, it falls back to
   installing the rendered formula file directly.

You can also use the workflow-dispatch path to republish assets for an
existing `v*` tag without creating a second tag. The workflow rejects branch
names or other non-tag refs.

Release archives intentionally derive packaged file mtimes from the tagged
commit timestamp in UTC, subject to ZIP's 2-second timestamp precision. That
keeps repeated builds of the same commit deterministic without making unpacked
files look like they came from `2000-01-01 00:00`.

## What Gets Published

- Prebuilt `darwin/amd64`, `darwin/arm64`, `linux/amd64`, and `linux/arm64`
  archives.
- A `SHA256SUMS` file for checksum verification.
- Tags with a prerelease suffix such as `-alpha.1` publish as GitHub
  prereleases rather than stable releases.
- The release binary reports the release version, build commit, and mode through
  `harness --version`.
- Archive entry timestamps are derived from the source commit time for the
  tagged revision, subject to ZIP's 2-second precision, rather than the
  wall-clock publish time.
- The default Homebrew formula `easyharness` tracks the current public release
  line: alpha today, stable later if stable tags are introduced.

## Homebrew Tap Publishing

Homebrew publishing uses the separate public repository
`catu-ai/homebrew-tap`. Because Homebrew lets users omit the `homebrew-`
prefix in tap commands, that repository is installed as `catu-ai/tap`.

Tagged releases update the tap on GitHub alone once these prerequisites are in
place:

1. Create `catu-ai/homebrew-tap` with an initial commit on its default branch.
   The workflow assumes that branch is `main`.
2. Add a repository secret named `EASYHARNESS_HOMEBREW_TAP_TOKEN` to
   `catu-ai/easyharness`.
3. Give that token contents-write access to `catu-ai/homebrew-tap`.

The release workflow renders `Formula/easyharness.rb` from the staged
`dist/release/SHA256SUMS` file after the GitHub Release assets are published,
then pushes the updated formula into the tap when the secret is available.

If the secret is missing, the release workflow emits a warning and skips the
tap update instead of blocking the archive upload. The repair path is:

1. Configure or fix `EASYHARNESS_HOMEBREW_TAP_TOKEN`.
2. Confirm `catu-ai/homebrew-tap` still has a writable default branch.
3. Re-run the Release workflow with `workflow_dispatch` for the same `v*` tag.

The formula name remains `easyharness`, while the installed binary remains
`harness`.

## Contributor Baseline

Release and CI jobs use the Go version recorded in `go.mod`, which is currently
`go 1.26.0`.
