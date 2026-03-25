# Releasing

`superharness` ships its first public alpha as GitHub Release archives built
from the tracked release workflow.

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

## Contributor Baseline

Release and CI jobs use the Go version recorded in `go.mod`, which is currently
`go 1.26.0`.
