# Agent Refactor

## What this directory is for

This directory is the working area for the current Agent refactor.

The purpose of this refactor is simple:

the project needs a smaller, clearer, and more stable Agent model before more Agent-related behavior is added.

The codebase already contains meaningful Agent behavior. What it still lacks is a sufficiently explicit and stable semantic boundary around that behavior.

This refactor exists to fix that first.

---

## Refactor stance

This is a maintenance-led consolidation effort.

It is not a general invitation to expand Agent behavior in parallel.

During this refactor window, Agent-related work should converge on the current refactor track instead of branching into new semantics.

That means:

- concept clarification before feature expansion
- boundary tightening before abstraction growth
- semantic consolidation before new behavior

---

## Core rule: minimum concepts only

This refactor follows one hard rule:

**do not introduce a new concept unless it is strictly necessary**

More explicitly:

- if an existing concept can be clarified, reuse it
- if an existing boundary can be made explicit, do that first
- if a behavior can be expressed without a new abstraction, do not add one
- "future flexibility" is not enough justification on its own

The goal of this refactor is not to grow the model.

The goal is to reduce ambiguity.

---

## What is being clarified

This refactor is currently concerned with the following questions:

1. what an `Agent` is
2. what an `AgentLoop` is
3. what the lifecycle of `AgentLoop` is
4. what the event surface around `AgentLoop` is
5. how persona / identity is assembled
6. how capabilities are represented
7. how context boundaries and compression work
8. how subagent coordination works

These are the current working boundaries.

If they need to be adjusted, they should be adjusted explicitly rather than drift implicitly in code.

---

## Status of this directory

The documents here are working materials.

They are not final or immutable.

If current notes are incomplete, incorrectly split, or too broad, they should be revised. This directory should evolve with the refactor rather than pretending the first draft is complete.

---

## Suggested document split

This directory may eventually contain notes such as:

- `agent-overview.md`
  - what an Agent is
- `agent-loop.md`
  - AgentLoop contract, lifecycle, event surface
- `persona.md`
  - persona and identity assembly
- `capability.md`
  - tools / skills / MCP capability semantics
- `context.md`
  - context scope, history, summary, compression
- `subagent.md`
  - subagent coordination rules

These files should be added only when they help clarify the current refactor work.

This directory should not turn into a generic architecture dump.

---

## What this directory is not for

This directory is not intended for:

- broad speculative architecture
- future multi-node protocol design not required by the current refactor
- parallel feature planning unrelated to Agent consolidation
- adding new concepts before current ones are made clear

If a topic does not directly help reduce ambiguity in the current Agent model, it probably does not belong here yet.

---

## Relationship to implementation

Implementation changes should not keep redefining Agent semantics implicitly.

If a PR changes or depends on Agent semantics, those semantics should either already exist here or be clarified in a linked issue first.

This directory is here to make implementation narrower and more disciplined.

---

## Relationship to GitHub tracking

The umbrella issue for this refactor should point here.

The issue is the coordination surface.

This directory is the repository-local working surface.

---

## Summary

The main question of this refactor is not:

- what more can Agent do

The main question is:

- what is the smallest stable model that current Agent behavior can be organized around
