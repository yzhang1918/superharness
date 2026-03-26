# microharness

`microharness` is a thin, agent-first harness CLI plus repository contract for
human-steered, agent-executed work.

GitHub home: [catu-ai/microharness](https://github.com/catu-ai/microharness)

The goal is to keep the harness legible and maintainable:

- tracked plans live in git
- runtime trajectory lives in `.local/`
- the CLI helps agents understand state and next actions
- skills teach agents how to run the workflow without a pile of fragile shell
  scripts

The repository is still in dogfood mode. The current cutover focuses on the
v0.2 command surface, command-owned evidence artifacts, and the canonical
`current_node` runtime model.

## Development Setup

Use the development installer to build a repo-local binary and expose
`harness` as a direct command:

```bash
scripts/install-dev-harness
```

By default the installer:

- builds the binary at `.local/bin/harness`
- installs a small worktree-aware `harness` wrapper in a user-local bin dir
- uses `~/.local/bin` by default, or `~/bin` when that is already on `PATH`
- keeps parallel worktrees isolated by dispatching to the current worktree's
  `.local/bin/harness`
- falls back outside `microharness` worktrees to the binary from the worktree
  that last installed the wrapper

Useful options:

```bash
scripts/install-dev-harness --help
scripts/install-dev-harness --install-dir "$HOME/.local/bin"
scripts/install-dev-harness --force
```

Verify the command is available:

```bash
command -v harness
harness --help
harness --version
```

After changing Go CLI code, rerun `scripts/install-dev-harness` so the direct
`harness` command stays in sync with the working tree.

If the installer reports that `harness` still resolves to a different binary,
either install into an earlier directory with `--install-dir` or move the
chosen install directory earlier in `PATH`.

## Public Alpha Release

The first public release is binary-first. External users should download the
archive that matches their platform from [GitHub
Releases](https://github.com/catu-ai/microharness/releases), verify the
published `SHA256SUMS`, and run the unpacked `harness` binary directly.

Supported alpha release targets are:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

Contributors should use the Go toolchain recorded in `go.mod`, which is
currently `go 1.26.0`.

Typical verification flow:

- macOS: `shasum -a 256 -c SHA256SUMS`
- Linux: `sha256sum -c SHA256SUMS`

Then unpack and inspect the release binary:

```bash
unzip microharness_<version>_<goos>_<goarch>.zip
cd microharness_<version>_<goos>_<goarch>
./harness --version
./harness --help
```

The release binary reports the release version, build commit, and mode. The
development installer remains available for contributors who are working from a
checkout.

## Current Command Surface

`microharness` currently ships these commands:

- `harness plan template`
- `harness plan lint`
- `harness execute start`
- `harness evidence submit`
- `harness status`
- `harness review start`
- `harness review submit`
- `harness review aggregate`
- `harness archive`
- `harness reopen --mode <finalize-fix|new-step>`
- `harness land --pr <url> [--commit <sha>]`
- `harness land complete`

The root CLI also exposes `harness --version` as a plain-text debug flag for
identifying the running binary. Unlike the stateful workflow commands above,
it is not a JSON-first command surface.

`harness ui` is deferred.

## Workflow

The repository currently uses this v0.2 workflow:

1. Discovery
2. Plan
3. Execute
4. Archive / publish / await merge approval
5. Land

`harness status` now resolves one canonical `state.current_node` rather than
reporting layered lifecycle or step-state fields. Common nodes include
`plan`, `execution/step-<n>/implement`, `execution/step-<n>/review`,
`execution/finalize/review`, `execution/finalize/archive`,
`execution/finalize/publish`, `execution/finalize/await_merge`, `land`, and
`idle`.

For medium or large work, create or update a tracked plan under
`docs/plans/active/`, execute against that plan, archive it under
`docs/plans/archived/` once the candidate is ready for local freeze, then
record publish, CI, and sync facts for the archived candidate through
`harness evidence submit` until status reaches
`execution/finalize/await_merge`. After merge, enter `land` with
`harness land --pr <url> [--commit <sha>]`, finish post-merge cleanup, then
run `harness land complete` so status returns to `idle`. If an archived
candidate becomes invalid before merge, reopen it with
`harness reopen --mode finalize-fix` for narrow repair or
`harness reopen --mode new-step` when the change deserves a new unfinished
step.

High-level guidance lives in [AGENTS.md](./AGENTS.md). The durable contracts
for plans and CLI behavior live in [docs/specs/index.md](./docs/specs/index.md).
Execution detail for agents lives in `.agents/skills/`.

## Repository Layout

- `cmd/harness/`: CLI entrypoint
- `internal/`: CLI implementation
- `docs/plans/`: tracked plans
- `docs/specs/`: durable repo contracts
- `.agents/skills/`: repo-local workflow skills
- `.local/harness/`: disposable runtime state, current-plan/last-landed
  markers, review artifacts, evidence artifacts, and trajectory

## Current Constraints

- one active review round at a time
- no web UI yet
- development installer remains available for contributors; GitHub Release
  packaging now exists for the public alpha binary
- Homebrew flow is still deferred
