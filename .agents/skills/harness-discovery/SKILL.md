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

## Explorer Subagent Decision

Use explorer subagents on demand, not by default.

- Keep user-supplied core context and other shared repository context with the
  controller whenever later questions may depend on the details.
- Stay local when the controller can answer the next question from the shared
  context it already needs to hold.
- Use one explorer subagent when one bounded repository question or hypothesis
  needs checking.
- Use multiple explorer subagents in parallel only when multiple bounded
  hypotheses or questions are genuinely independent.
- Do not split one shared context bundle across multiple explorer subagents
  just to get summaries back.
- Explorer subagents should return factual findings for the bounded question
  only. They do not choose the next user question, recommend the workflow
  direction, or replace controller judgment.

## Execution Contract

1. If the task is still fuzzy, ask one concise clarification question before
   doing broader discovery.
2. Read the most relevant repository context needed to ask sharper questions.
3. Use bounded repository exploration according to `Explorer Subagent
   Decision` above whenever local reading alone is not enough.
4. Discovery may alternate between human answers and further bounded
   exploration. Re-evaluate whether more exploration is needed after each
   clarification turn.
5. Ask exactly one high-leverage question per turn.
6. Use Socratic questioning to clarify:
   - purpose
   - constraints
   - non-goals
   - success criteria
   - workflow direction
7. When a decision benefits from framing, present 2-4 realistic options.
8. For each option, give:
   - a short label
   - one clear upside
   - one clear downside
   - when it fits
9. Recommend a direction when the tradeoffs are asymmetric.
10. Converge on a concrete approach, draft acceptance criteria, and state the
   next workflow step explicitly.
11. Hand off to `harness-plan` only after the human confirms the direction.

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
- Do not treat explorer use as mandatory when local reading is enough.
- Do not let explorer subagents own the shared context the controller still
  needs for later questioning.
- Do not treat factual explorer output as permission to skip controller
  synthesis or user clarification.
- Do not proceed until the human has enough clarity to approve the next step.
- Do not turn discovery into a hidden plan that only exists in chat.
