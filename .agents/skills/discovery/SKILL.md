---
name: discovery
description: Use when starting medium or large work in superharness and you need to clarify goals, constraints, tradeoffs, or whether the repo needs a new plan before implementation.
---

# Discovery

## Purpose

Use discovery to turn a request into a clear execution target before writing or
changing a tracked plan.

## When to Use

- a new request needs scope clarification
- the next step is unclear or has non-obvious tradeoffs
- archived work may need to be reopened but the right path is uncertain
- human feedback suggests a real direction change rather than a small fix

## Workflow

1. Confirm the objective, success criteria, and important constraints.
2. Decide whether this is:
   - a new plan
   - a change to the current active plan
   - a reopen of an archived plan
   - a small direct fix that does not need a new discovery pass
3. Surface the main tradeoffs one at a time instead of asking for a giant dump
   of decisions.
4. Record only durable conclusions. Discovery itself should not create tracked
   implementation files.
5. Hand off to the `plan` skill once scope is clear enough to write or update a
   tracked plan.

## Exit Criteria

Discovery is done when you can state:

- what problem this slice solves
- what is intentionally out of scope
- what success looks like
- whether execution should continue through a new plan or a reopened one

## Commands

- Use `harness status` when you need to understand the current plan and local
  workflow state before deciding whether to reopen or continue.

## Do Not

- Do not start implementation during discovery.
- Do not create extra top-level workflow states beyond the repo lifecycle.
- Do not turn discovery into a hidden plan written only in chat.

