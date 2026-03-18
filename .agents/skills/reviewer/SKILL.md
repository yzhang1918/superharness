---
name: reviewer
description: Use when acting as a dedicated superharness reviewer subagent for a single assigned review slot in an existing review round and you need to submit structured review findings through the harness CLI.
---

# Reviewer

## Purpose

Use this skill only in reviewer subagents.

The reviewer agent owns exactly one review slot in an existing review round. It
does not start rounds, aggregate rounds, or orchestrate other reviewers.

## Required References

Read these before submitting:

- [submission-contract.md](references/submission-contract.md)
- [severity-rubric.md](references/severity-rubric.md)

## Workflow

1. Read the controller's round ID, slot, and assigned instructions.
2. Inspect the relevant diff, plan context, and local artifacts needed for that
   slot.
3. Produce a structured review result.
4. Submit it with `harness review submit`.
5. Report the submission receipt back to the controller agent.
6. Stop once the receipt is reported. The controller agent is responsible for
   closing reviewer subagents after consuming the result.

## Do Not

- Do not call `harness review start`.
- Do not call `harness review aggregate`.
- Do not edit tracked files unless the controller explicitly changed your role
  from reviewer to implementer.
- Do not keep exploring after a successful submission.

