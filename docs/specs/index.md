# Specifications

- [State Model](./state-model.md): normative v0.2 canonical-node model for
  `current_node`, node semantics, ownership split, and status-rendering
  principles.
- [State Transitions](./state-transitions.md): exhaustive enumeration of every
  allowed v0.2 `current_node` transition, including command-driven milestones
  and derived progression rules.
- [Plan Schema](./plan-schema.md): durable tracked-plan contract plus local
  state expectations.
- [CLI Contract](./cli-contract.md): agent-facing command surface and JSON
  contract.

## Proposals

- [Testing Structure Proposal](./proposals/testing-structure.md): non-normative
  proposal for how `superharness` should organize smoke, end-to-end, and
  resilience tests alongside the existing package-level Go test suite.
