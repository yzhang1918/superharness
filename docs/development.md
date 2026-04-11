# Development

This repository is developed primarily through agents. For repo-specific
operating rules, start with [AGENTS.md](../AGENTS.md). This document keeps the
longer local setup and maintainer details that do not belong on the root
README.

## Local Harness Setup

Use the development installer to build a repo-local binary and expose
`harness` as a direct command:

```bash
scripts/install-dev-harness
```

By default the installer:

- builds the binary at `.local/bin/harness`
- installs a small worktree-aware `harness` wrapper in a user-local bin dir
- uses `~/.local/bin` by default
- keeps parallel worktrees isolated by dispatching to the current worktree's
  `.local/bin/harness`
- dispatches outside easyharness source trees to a stable `harness` already on
  `PATH`, such as a Homebrew install

Useful options:

```bash
scripts/install-dev-harness --help
scripts/install-dev-harness --install-dir "$HOME/.local/bin"
scripts/install-dev-harness --force
```

Development installs expect `~/.local/bin` to be on `PATH` so the wrapper can
be called directly:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Inside any easyharness source tree, the wrapper still dispatches to that
checkout's local `.local/bin/harness` and does not silently fall back to a
stable Homebrew or other PATH-installed binary.

Outside easyharness source trees, the wrapper expects a stable `harness`
installation to already be available on `PATH`. The normal release path is the
Homebrew install shown in the root [`README.md`](../README.md).

Verify the command is available:

```bash
command -v harness
harness --help
harness --version
```

After changing Go CLI code, rerun `scripts/install-dev-harness` so the direct
`harness` command stays in sync with the working tree.

Contributors should use the Go toolchain recorded in `go.mod`, which is
currently `go 1.25.0`.

## UI Development

When changing the embedded UI shell under `web/`, rebuild the production UI
assets before relying on `harness ui` or rerunning Go builds/tests that embed
the UI:

```bash
pnpm --dir web install
pnpm --dir web build
```

For browser-level validation of the embedded shell, use the repo helper scripts
that drive the local UI through the bundled Playwright wrapper:

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

Or point Vite at the actual `harness ui` URL explicitly when you prefer the
CLI default auto-selected port:

```bash
harness ui --no-open
HARNESS_UI_API_TARGET=http://127.0.0.1:<actual-port> pnpm --dir web dev
```

## Bootstrap Asset Editing

This repository dogsfoods the same bootstrap assets that `harness init` and
the bootstrap resource commands package for other repositories.

Edit `assets/bootstrap/` when changing the harness-managed skill pack or the
managed `AGENTS.md` block content. Treat `.agents/skills/` in this repository
as tracked materialized output from `assets/bootstrap/`, not as a hand-edited
source tree.

After editing `assets/bootstrap/`, refresh the generated outputs with:

```bash
scripts/sync-bootstrap-assets
scripts/sync-bootstrap-assets --check
```

If the installer reports that `harness` still resolves to a different binary,
either install into an earlier directory with `--install-dir` or move the
chosen install directory earlier in `PATH`.
