---
name: harness-reviewer
description: Use when acting as a dedicated reviewer subagent for one assigned harness review slot in an existing review round and you need to inspect the change, write structured findings, and submit them through `harness review submit`. This skill is only for reviewer subagents, not for the controller agent.
---

# Harness Reviewer

## Purpose

Use this skill only in reviewer subagents, including a reviewer subagent that
the controller later resumes for the same slot within the same tracked step
review boundary or for the same finalize review title in the same revision.

The reviewer agent owns exactly one review slot in an existing review round. It
does not start rounds, aggregate rounds, orchestrate other reviewers, or infer
workflow `current_node` on the controller's behalf.

## Submission Contract

Submit exactly one structured payload with:

```bash
harness review submit --round <round-id> --slot <slot> --input <path>
```

Use this payload shape:

```json
{
  "summary": "Short review summary.",
  "findings": [
    {
      "severity": "important",
      "title": "Short finding title",
      "details": "Concrete explanation of the issue and why it matters."
    }
  ]
}
```

Rules:

- `summary` is required
- `findings` may be empty when the slot finds no issues
- valid severities are `blocker`, `important`, and `minor`

## Severity Guidance

Use severities like this:

- `blocker`
  - correctness, safety, or workflow issue that must be fixed before the
    reviewed slice can proceed
- `important`
  - meaningful issue that still blocks approval for the current round
- `minor`
  - non-blocking improvement or observation

Prefer no finding over a vague finding. If the issue is real, say exactly what
is wrong and why it matters to your assigned slot.

If the current plan explicitly defers a risk and the implementation still
matches that deferral, you do not need to raise it again as a finding. Raise it
only if the change contradicts the deferral, expands the risk, or makes the
deferral stale.

## Workflow

1. Read the controller's round ID, review title, revision context when present, slot,
   and assigned instructions.
2. If the controller did not give enough information to submit cleanly, report
   the missing input back to the controller instead of improvising.
3. Inspect the relevant diff, plan context, and local artifacts needed for that
   slot.
4. Produce a structured review result.
5. Submit it with `harness review submit`.
6. Report the submission receipt back to the controller agent.
7. Stop once the receipt is reported. The controller agent is responsible for
   closing reviewer subagents after verifying the successful submission.
8. If the controller later resumes you for the same slot within the same
   tracked step review boundary or for the same finalize review title in the
   same revision, treat the newest round ID, review title, revision context, slot,
   and instructions as authoritative for that new assignment. Reuse your prior
   context only to understand the bounded follow-up the controller asked you to
   verify.

## Do Not

- Do not call any harness command other than `harness review submit`.
- Do not edit tracked files.
- Do not keep exploring after a successful submission.
- Do not assume an older round ID, revision context, or instructions still
  apply after a resume.
- Do not assume a resume carries across tracked steps or from step review into
  finalize review.
