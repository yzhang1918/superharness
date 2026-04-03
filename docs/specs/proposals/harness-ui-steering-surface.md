# Harness UI Steering Surface Proposal

## Status

This document is a non-normative proposal.

It describes a recommended direction for a future `harness ui` surface. It
does not change the current CLI, plan, or state contracts by itself.

## Background

Two open issues define the current UI problem space:

- [Issue #2](https://github.com/catu-ai/easyharness/issues/2): add `harness ui`
  for local status and trajectory visualization
- [Issue #70](https://github.com/catu-ai/easyharness/issues/70): define the
  human steering surface for harness UI and status

Several interaction principles are already clear enough to state directly:

- dense, IDE-like layouts work better than dashboard cards
- humans need one-page visibility more than decorative summaries
- `next actions` and `why now` matter more than raw state dumps
- file trees, diffs, and review artifacts need a document-style viewer
- a global deck of all possible actions becomes noisy and hard to steer from

This proposal turns those principles into a real `harness ui` direction for
the current repository.

## Purpose

`harness` already has a strong agent-facing CLI and durable tracked/local
artifact model. The missing piece is a human-facing steering surface that lets
someone quickly answer:

- what state is the work in now
- what changed last
- what review evidence exists
- what is the next meaningful action
- where human judgment is needed

The UI should complement `harness status`, not replace it.

## Goals

- provide a clear human steering surface for the current worktree
- stay grounded in existing tracked files and `.local/harness` runtime
  artifacts
- make `next actions`, blockers, and review state easy to understand
- show plans, tracked diffs, review artifacts, and recent trajectory in one
  dense local workbench
- minimize full-page scrolling and avoid multi-screen dashboard sprawl
- keep the product visually calm, technical, and legible under sustained use

## Non-Goals

- introducing a second hidden state source alongside tracked and local
  artifacts
- turning `harness` into a human-only workflow that bypasses the CLI and
  repository contract
- shipping a general-purpose IDE or repository browser unrelated to harness
  steering
- exposing every possible command as a permanent always-visible action deck

## Summary

The recommended product shape is a local `harness ui` workbench for the
current repository. It should feel closer to VS Code than to a dashboard:

- narrow top bar
- icon-only activity rail on the left
- explorer pane in the middle
- editor/detail pane on the right
- collapsible bottom status drawer

The UI should read from `harness status`, tracked plan files, git diff, and
`.local/harness` artifacts. It should present those sources through a dense,
document-oriented interface instead of inventing new product-only state.

## Product Shape

### Entry Point

`harness ui` should start a local server for the current working repository and
open a local workbench surface.

It should inspect the actual current worktree and its harness artifacts rather
than a disposable demo workspace.

### Default Role

The first version of the UI should be read-first and steer-first, not
action-first.

The user should primarily understand state, inspect evidence, and decide what
the agent should do next. Direct UI-triggered actions may exist, but they
should be contextual and sparse rather than the main organizing principle.

### Relationship to `harness status`

`harness status` remains the fast, scriptable, agent-facing summary.

`harness ui` should provide:

- denser context
- artifact navigation
- richer review visibility
- latest-change inspection
- clearer human steering moments

`harness ui` should not fork a second interpretation of workflow state. When
possible, it should render the `harness status` view more richly rather than
reinterpret it from scratch.

## Design Principles

### One Page, Not a Dashboard

The main workbench should fit the important information on one screen:

- current status
- next actions
- changed files or active review evidence
- recent trajectory

The user should navigate panes, tabs, and drawers, not scroll through stacked
marketing-style sections.

### Dense and Quiet

The UI should avoid:

- large hero headers
- decorative card grids
- rounded-box overload
- mixed font systems
- oversized labels that repeat the obvious

It should prefer:

- thin separators
- restrained spacing
- consistent type
- clear contrast
- utilitarian visual hierarchy

### Contextual Actions, Not an Action Catalog

A full list of ready and blocked actions quickly turns into noise.

The real UI should emphasize only the few actions that matter now. Blocked
actions may be inspectable, but they should not dominate the default view.

### Artifact-First Steering

Humans steer better when they can inspect evidence directly:

- the tracked plan
- the latest diff
- review findings
- aggregate review results
- evidence artifacts
- recent command trajectory

The UI should always lead back to those artifacts.

## Visual Direction

The recommended visual direction is a dark, border-light workbench inspired by
VS Code.

### Carry Forward

- icon-only activity bar
- resizable middle and right panes
- collapsible bottom drawer for status
- tree-style file explorer
- document tabs for file contents, diffs, manifests, ledgers, and history
  payloads
- unified UI typography with monospace reserved for code, diffs, and raw
  documents

### Avoid

- a giant branded heading at the top of the page
- large rounded cards for every content block
- repeated explanatory prose in the main working area
- separate "raw JSON" boxes as a primary surface
- wide action decks that list many blocked options by default

## Information Architecture

### Activity Rail

The left rail should use icons only. Recommended sections:

- Overview
- Reviews
- Timeline
- Settings

The rail should stay narrow and visually secondary.

### Overview

Overview is the main steering entry.

It should show:

- status summary
- current node
- next actions
- blockers and warnings
- current step or current review target
- the latest meaningful change or event

This view should answer "What is happening, and what should the human decide
now?" before the user opens any deeper pane.

### Reviews

Reviews should make review state first-class.

It should show:

- active review round, if any
- review kind and target
- findings by severity
- manifest, ledger, aggregate, and submissions as document tabs
- whether the review is waiting for submissions, aggregation, fixes, or human
  judgment

This area is critical for issue #70 because review evidence is one of the main
human steering inputs.

### Timeline

Timeline should present the recent harness trajectory in a human-readable form.

Each run should capture:

- label
- time
- kind
- full command when applicable
- generated input documents
- resulting output documents
- tracked diff for patch-style transitions

This keeps recent history inspectable without forcing humans to reconstruct it
from raw files alone.

### Settings

Settings should hold low-frequency controls and environment details.

Examples:

- refresh
- local server/session info
- repository/worktree path
- debug or raw inspection affordances

These controls should not live in the main header.

## Layout

### Top Bar

The top bar should be thin and mostly infrastructural:

- repository or worktree path
- connection or freshness indicator
- perhaps a concise title

It should not contain large branding or duplicate state already visible in the
status drawer.

### Explorer and Editor Panes

The middle explorer and right editor panes should be independently scrollable
and resizable.

The explorer is for selection.

The editor is for inspection.

This separation worked well in the earlier prototype and should remain a core
interaction model.

### Bottom Status Drawer

The bottom drawer should be default-open on first load and collapsible after
that.

It should contain:

- current node
- status summary
- next actions
- blockers
- key facts
- warnings

This is the one place where the user should always be able to recover the
current harness state quickly.

## Steering Model

### Passive Visibility

Most workflow moments only need visibility:

- progress through steps
- what changed
- what review artifacts exist
- whether archive or merge-readiness blockers remain

The UI should make these moments easy to inspect without pushing the user
toward unnecessary clicks.

### Explicit Human Input

Some workflow moments should surface explicit human steering more clearly:

- approve or reject a branch candidate for archive
- decide whether reopened work is narrow repair or a new step
- interpret blocking review findings
- decide whether work should continue, redirect, or stop

These moments should appear as contextual decisions near the relevant status
and evidence, not as a permanent global action catalog.

## Data Sources

The UI should read from durable or already-owned sources:

- `harness status`
- tracked plan files under `docs/plans/`
- tracked git state and diff
- `.local/harness/current-plan.json`
- review artifacts
- evidence artifacts
- other harness-owned local metadata

It should avoid introducing a second long-lived product-only database or hidden
state model.

Ephemeral UI state is acceptable for:

- pane sizes
- selected tab
- collapse state
- local session preferences

## Interaction Patterns to Preserve

Several interaction patterns are worth preserving:

- IDE-style workbench layout
- bottom status drawer
- proper file tree instead of flat file lists
- history entries with document tabs
- separation between tracked diff and local runtime artifacts
- contextual artifact inspection rather than walls of explanatory prose

## Patterns to Avoid

- a matrix of all possible synthetic actions
- visually heavy header and card-based layout
- product energy spent on explaining the UI instead of helping a human steer

## Proposed Phasing

### Phase 1: Read-First Steering Workbench

Ship:

- top-level `harness ui`
- Overview, Reviews, Timeline, Settings
- status drawer
- artifact inspection grounded in harness-owned data, not a general-purpose IDE
  browser

Defer:

- direct action triggers beyond minimal refresh or open behaviors

### Phase 2: Contextual Steering Actions

Add:

- explicit approval and redirect affordances at the workflow moments that need
  human choice
- deeper review triage interactions
- clearer archive and merge-readiness handoff actions

### Phase 3: Remote Signal Integration

Extend the workbench when remote facts become first-class:

- PR state
- CI freshness
- merge readiness
- sync drift

This phase should build on the same steering surface rather than creating a
separate dashboard.

## Open Questions

- which steering actions truly belong in the UI versus staying CLI-only
- whether Overview and Timeline should be separate primary sections or one
  combined steering view
- how much remote PR and CI state should appear before the contracts for those
  signals settle
- whether review findings need their own dedicated left-rail section beyond
  Reviews

## Recommendation

Treat issue #70 as the product-definition frame and issue #2 as the delivery
vehicle.

Build `harness ui` as a dense local steering workbench for the current repo,
not as a dashboard. Preserve the artifact-first, status-grounded model, and
carry forward the strongest interaction patterns outlined above:

- VS Code-like workbench structure
- one-page density
- strong status and next-action visibility
- file, diff, and review artifact inspection as first-class tasks
- contextual human steering instead of a giant action deck
