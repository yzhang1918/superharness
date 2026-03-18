# Severity Rubric

Use severities the way the CLI expects them:

- `blocker`
  - correctness, safety, or workflow issue that must be fixed before the
    reviewed slice can proceed
- `important`
  - meaningful issue that still blocks approval for the current round
- `minor`
  - non-blocking improvement or observation

In the current CLI contract, both `blocker` and `important` are treated as
blocking during aggregation.

Prefer no finding over a vague finding. If the issue is real, say exactly what
is wrong and why it matters to the assigned review slot.

