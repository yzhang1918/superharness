# Submission Contract

Reviewer submissions are structured and deterministic.

Use:

```bash
harness review submit --round <round-id> --slot <slot> --input <path>
```

You may also pipe JSON through stdin. Check `harness review submit --help` if
you need the exact invocation details.

## Submission Shape

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

After a successful submit:

1. send the receipt back to the controller agent
2. state that submission is complete
3. stop so the controller can close the reviewer agent

