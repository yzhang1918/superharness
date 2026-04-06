# easyharness

`easyharness` is a thin, git-native harness CLI plus repository contract for
human-steered, agent-executed work.

GitHub home: [catu-ai/easyharness](https://github.com/catu-ai/easyharness)

The project is named `easyharness`. The CLI executable remains `harness`.

The goal is to keep the harness legible and maintainable:

- standard plans live in git
- lightweight low-risk active plans still live in git
- lightweight archived snapshots live in `.local/harness/`
- runtime trajectory lives in `.local/`
- the CLI helps agents understand state and next actions
- skills teach agents how to run the workflow without a pile of fragile shell
  scripts

The repository is still in dogfood mode. The current cutover focuses on the
v0.2 command surface, command-owned evidence artifacts, and the canonical
`current_node` runtime model.

`easyharness` is also in a rapid development phase, so external users should
expect breaking changes between releases. Compatibility guarantees and
migration support are not the current priority.

Field-level contract artifacts now live in:

- `schema/index.json` for the checked-in JSON Schema registry
- `docs/specs/contract.md` for the normative guide to what that registry covers

The field-level source of truth is the Go-owned contract module under
`internal/contracts/`. We intentionally do not render one markdown page per
schema because that was mostly duplicating the schema files themselves.
`docs/specs/contract.md` also distinguishes the stable public command surface
from CLI-owned runtime artifacts such as `.local/harness/*`.
Refresh or verify the checked-in registry with:

```bash
scripts/sync-contract-artifacts
scripts/sync-contract-artifacts --check
```

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
- falls back outside `easyharness` worktrees to the binary from the worktree
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

When changing the embedded UI shell under `web/`, rebuild the production UI
assets before relying on `harness ui` or rerunning Go builds/tests that embed
the UI:

```bash
pnpm --dir web install
pnpm --dir web build
```

For browser-level validation of the embedded shell, use the repo helper that
drives the local UI through the
[$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) wrapper:

```bash
scripts/ui-playwright-smoke
scripts/ui-playwright-review-smoke
```

Use `scripts/ui-playwright-smoke` for the general shell, rail, and archived-plan
browser path. Use `scripts/ui-playwright-review-smoke` whenever the `Review`
page changes, or when you want the populated round-browser validation that
exercises active-plan review data, degraded review artifacts, and review-only
states such as empty active plans.

For frontend development against the live backend, run the bundled backend dev
command in one terminal so Vite's default `/api` proxy has a live target on
`127.0.0.1:4310`, then start Vite in a second terminal:

```bash
pnpm --dir web dev:harness
pnpm --dir web dev
```

or point Vite at the actual `harness ui` URL explicitly when you prefer the
CLI default auto-selected port:

```bash
harness ui --no-open
HARNESS_UI_API_TARGET=http://127.0.0.1:<actual-port> pnpm --dir web dev
```

When changing the harness-managed bootstrap contract that this repository
dogsfoods, edit `assets/bootstrap/` instead of hand-editing `.agents/skills/`.
The `.agents/skills/` tree and this repository's managed `AGENTS.md` block are
tracked materialized outputs of those packaged bootstrap assets. Refresh them
with:

```bash
scripts/sync-bootstrap-assets
scripts/sync-bootstrap-assets --check
```

If the installer reports that `harness` still resolves to a different binary,
either install into an earlier directory with `--install-dir` or move the
chosen install directory earlier in `PATH`.

## Public Alpha Release

The public alpha remains GitHub Release-backed. External users can either
install `easyharness` from the dedicated Homebrew tap or download a release
archive directly from [GitHub
Releases](https://github.com/catu-ai/easyharness/releases). In both cases, the
installed executable is still named `harness`.

Supported alpha release targets are:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

Contributors should use the Go toolchain recorded in `go.mod`, which is
currently `go 1.25.0`.

Typical verification flow:

- macOS: `shasum -a 256 -c SHA256SUMS`
- Linux: `sha256sum -c SHA256SUMS`

Homebrew install flow:

```bash
brew tap catu-ai/tap
brew install catu-ai/tap/easyharness
harness --version
```

First-run repository bootstrap after installing the binary:

```bash
cd /path/to/your-repo
harness install --dry-run
harness install
```

`harness install` writes the minimum harness-managed repository contract for a
repo: a managed block inside `AGENTS.md` plus the repo-local skill pack under
`.agents/skills/`. The command is safe to rerun after upgrades. Repeated runs
either refresh the known managed assets in place or report a no-op when the
repository is already current. User-owned `AGENTS.md` content outside the
managed block is preserved.

Upgrade a Homebrew install with:

```bash
brew update
brew upgrade catu-ai/tap/easyharness
```

The default Homebrew formula currently tracks the public alpha release line.
If `easyharness` later starts shipping stable tags, the same default formula
will move to the stable line rather than keeping alpha on a separate package
name.

If you prefer to inspect the release archive directly, unpack and inspect the
binary:

```bash
unzip easyharness_<version>_<goos>_<goarch>.zip
cd easyharness_<version>_<goos>_<goarch>
./harness --version
./harness --help
```

The release binary reports the release version, build commit, and mode. The
development installer remains available for contributors who are working from a
checkout.

Maintainers cut releases from a dedicated release PR that updates the root
`VERSION` file, plus any related release docs. `VERSION` stores the unprefixed
release version such as `0.1.0-alpha.6`; after that PR merges to `main`,
automation creates the matching `v*` tag and dispatches the existing `Release`
workflow, which then publishes the release assets and Homebrew formula updates
for that tag.

## Current Command Surface

`easyharness` currently ships these commands:

- `harness plan template`
- `harness plan lint`
- `harness install`
- `harness execute start`
- `harness evidence submit`
- `harness status`
- `harness ui`
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

`harness ui` starts a local read-only workbench for the current repository.
The currently delivered slices expose live `Status`, `Timeline`, and `Review`
pages. `Timeline` is backed by the current plan's command-owned event index,
and `Review` renders the active plan's read-only review rounds. The UI keeps
an IDE-like steering surface, but it intentionally does not duplicate a
general-purpose file browser or diff viewer that is already better served by
external IDEs.

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

For medium or large work, or any change that is not explicitly eligible for
the lightweight path, create or update a tracked standard plan under
`docs/plans/active/`, execute against that plan, archive it under
`docs/plans/archived/` once the candidate is ready for local freeze, then
record publish, CI, and sync facts for the archived candidate through
`harness evidence submit` until status reaches
`execution/finalize/await_merge`. After merge, enter `land` with
`harness land --pr <url> [--commit <sha>]`, finish post-merge cleanup, then
run `harness land complete` so status returns to `idle`.

For narrow low-risk work, `harness` may instead use a tracked active plan under
`docs/plans/active/` with the same schema plus `workflow_profile: lightweight`.
The lightweight path reuses the same canonical nodes as standard work, but on
archive it writes the archived snapshot to
`.local/harness/plans/archived/<plan-stem>.md` instead of
`docs/plans/archived/`. Lightweight work still requires human steering, must
stay explicitly in-bounds, and must leave a small repo-visible breadcrumb such
as a PR body note explaining why the lightweight path was used. If any
lightweight candidate stops looking low-risk, it should escalate back to the
standard tracked-plan path.

Use `lightweight` only when all of these are true:

- the whole slice is one bounded low-risk maintenance change
- the edits are limited to README/docs/comments/copy or similarly
  non-behavioral cleanup
- no `harness` behavior, normative spec, state rule, persistence behavior,
  release or CI workflow, or security-sensitive logic changes
- if the boundary is unclear, default to `standard`

In practice, lightweight is for tiny bounded low-risk changes such as README
wording, doc clarification, comment cleanup, or similarly narrow non-behavioral
metadata fixes. If the change touches CLI behavior, runtime state, review or
archive semantics, release flow, or any normative contract meaning, use the
standard tracked-plan path instead.

If an archived candidate becomes invalid before merge, reopen it with
`harness reopen --mode finalize-fix` for narrow repair or
`harness reopen --mode new-step` when the change deserves a new unfinished
step. If a repository has not been bootstrapped yet, run `harness install`
first so the managed `AGENTS.md` block and repo-local skills exist before the
workflow starts.

High-level guidance lives in [AGENTS.md](./AGENTS.md). The durable contracts
for plans and CLI behavior live in [docs/specs/index.md](./docs/specs/index.md).
Execution detail for agents is materialized into `.agents/skills/` from the
canonical bootstrap assets under `assets/bootstrap/`.

## Repository Layout

- `cmd/harness/`: CLI entrypoint
- `internal/`: CLI implementation
- `assets/bootstrap/`: canonical source for packaged bootstrap assets that this
  repository dogsfoods
- `docs/plans/`: tracked active plans for both profiles plus archived standard plans
- `docs/specs/`: durable repo contracts
- `.agents/skills/`: tracked materialized repo-local workflow skills generated
  from `assets/bootstrap/`
- `AGENTS.md`: repo-specific guidance plus the harness-managed install block
- `.local/harness/`: disposable runtime state, current-plan/last-landed
  markers, archived lightweight plan snapshots, review artifacts, evidence
  artifacts, and trajectory

## Current Constraints

- one active review round at a time
- no web UI yet
- development installer remains available for contributors; GitHub Release
  packaging now exists for the public alpha binary
- Homebrew tap publishing depends on the tap repo and cross-repo token being
  configured for tagged releases
