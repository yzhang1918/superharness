# Project Naming Proposal: `easyharness`

## Status

This document is a non-normative proposal.

It records the current naming recommendation for the project. It does not
rename the repository, Go module path, release assets, or live docs by itself.

## Purpose

`microharness` already reflects an intentional product shape: thin,
agent-first, legible, and maintainable. The recent rename from
`superharness` to `microharness` improved that engineering fit, but the name
still reads like an implementation stance more than a user-facing promise.

This proposal captures the current recommendation from a user-mental-model
perspective, assuming migration cost is temporarily ignored so the naming
decision can be evaluated on clarity and product positioning alone.

## Recommendation

If the project is renamed again, prefer `easyharness` over both
`microharness` and `superharness`.

The reasoning is:

- `easyharness` communicates the intended user outcome on first read.
- `microharness` communicates internal shape and engineering taste more than
  direct user value.
- `superharness` over-signals breadth or power and clashes with the current
  thin, maintainable product direction.

## Goals

- choose the most understandable project name for first-time users
- keep the `harness` executable name unchanged
- preserve room for the project to cover planning, execution, review, archive,
  and landing flows rather than a single narrow feature
- make the naming rationale durable so a future agent does not have to
  reconstruct it from chat history

## Non-Goals

- deciding whether the rename should happen immediately
- estimating the operational migration cost in detail
- changing the CLI command from `harness` to another executable name
- rewriting archived plans or historical materials solely to erase old names

## User-Mental-Model Comparison

### `easyharness`

What users infer:

- this tool makes agent-driven work easier to run
- the value is onboarding, legibility, and lower friction

Strengths:

- clearest first impression
- easiest to say and remember
- strongest fit for README headlines, release notes, and word-of-mouth

Weaknesses:

- sounds more like a product promise than a technical posture
- risks feeling generic unless the supporting tagline stays specific

### `microharness`

What users infer:

- this tool is small, thin, or minimal
- the value is implementation style or architecture discipline

Strengths:

- stronger engineering personality
- fits the current "thin, legible, maintainable" positioning well

Weaknesses:

- less immediate for first-time users
- does not clearly state the benefit of using the project

### `superharness`

What users infer:

- this tool is broad, powerful, or maximal
- the value is capability volume rather than simplicity

Strengths:

- energetic and easy to notice

Weaknesses:

- mismatched with the current thin-contract product direction
- easier to read as hype than as a credible project promise

## Positioning Implications

If the project adopts `easyharness`, the surrounding language should keep the
name anchored to the actual product shape rather than a vague "easy AI" claim.

Recommended supporting lines include:

- `easyharness`: a thin, git-native harness for human-steered, agent-run work
- `easyharness`: make coding-agent workflows easier to run, review, and trust

Those lines preserve the clarity of `easyharness` while still signaling that
the project is opinionated about tracked plans, disposable local trajectory,
and evidence-first review flow.

## Decision Summary

From a pure user-mental-model standpoint:

1. `easyharness` is the strongest candidate.
2. `microharness` remains a solid engineering-facing fallback.
3. `superharness` should not be revived.

If the repository later chooses to pursue another rename, the default plan
should be to rename toward `easyharness` while explicitly keeping the
`harness` binary name stable.
