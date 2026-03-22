---
name: harness-discovery
description: Run interactive, Socratic pre-implementation discovery for medium/large or ambiguous work in a harness-driven repository by clarifying goals, constraints, tradeoffs, and workflow direction before planning or execution. Use this whenever the next move is unclear, the user needs help choosing an approach, or archived work may need to reopen.
---

# Harness Discovery

## Overview

Run discovery before implementation when the task needs real clarification.
Discovery is conversation-only. It should reduce ambiguity, surface tradeoffs,
and end with a clear next workflow step.

## Inputs

- the human's objective or problem statement
- relevant plans, specs, or design context from the repository
- current `harness status` output when the repository already has an active
  plan and local state

## Execution Contract

1. If the task is still fuzzy, ask one concise clarification question before
   doing broader discovery.
2. Read the most relevant repository context needed to ask sharper questions.
   Be comprehensive enough to avoid shallow questions, but do not disappear
   into aimless exploration.
3. Ask exactly one high-leverage question per turn.
4. Use Socratic questioning to clarify:
   - purpose
   - constraints
   - non-goals
   - success criteria
   - workflow direction
5. When a decision benefits from framing, present 2-4 realistic options.
6. For each option, give:
   - a short label
   - one clear upside
   - one clear downside
   - when it fits
7. Recommend a direction when the tradeoffs are asymmetric.
8. Converge on a concrete approach, draft acceptance criteria, and state the
   next workflow step explicitly.
9. Hand off to `harness-plan` only after the human confirms the direction.

## Option Framing Pattern

When you offer options, keep them concise and decision-shaped. A good pattern
is:

1. `Option A`
   - upside
   - downside
   - best when ...
2. `Option B`
   - upside
   - downside
   - best when ...
3. `Option C`
   - upside
   - downside
   - best when ...

Then add a short recommendation and why.

## Output

Discovery should end with a concise conversation summary containing:

- the problem statement
- key constraints and non-goals
- the accepted direction
- rejected alternatives with short rationale
- draft acceptance criteria
- the next workflow step

## Guardrails

- Do not implement code in this skill.
- Do not write or modify repository files during discovery.
- Do not ask bundled multi-question prompts; keep one question per turn.
- Do not offer weak filler options just to reach four.
- Do not turn option framing into long compare tables or verbose essays.
- Do not proceed until the human has enough clarity to approve the next step.
- Do not turn discovery into a hidden plan that only exists in chat.
