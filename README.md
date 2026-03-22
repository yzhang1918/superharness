# superharness

`superharness` is a thin, agent-first harness CLI plus repository contract for
human-steered, agent-executed work.

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
- links `harness` into the first writable directory already on `PATH`
- falls back to `~/.local/bin` when no writable `PATH` directory is available

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
```

After changing Go CLI code, rerun `scripts/install-dev-harness` so the direct
`harness` command stays in sync with the working tree.

If the installer reports that `harness` still resolves to a different binary,
either install into an earlier directory with `--install-dir` or move the
chosen install directory earlier in `PATH`.

## Current Command Surface

`superharness` currently ships these commands:

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
- development install only; no release packaging or Homebrew flow yet
