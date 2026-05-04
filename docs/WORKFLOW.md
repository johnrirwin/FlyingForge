# AI-Augmented Product Development Workflow

A practical guide to building products with AI assistance across the full development lifecycle while keeping the workflow portable across tools and vendors.

---

## Overview

This workflow uses working software as the source of truth and applies AI help at each stage of product development.

### Core Principle

> Code documents real states, interactions, and edge cases better than static prose alone.

### Subagent Principle

Default to subagents for research-heavy or exploratory work. If multiple questions can be investigated independently—such as product research, codebase discovery, risk analysis, option comparison, or root-cause investigation—run them in parallel via subagents and merge the results before planning or implementation.

```text
Vision → Prototype → Requirements → Tickets → Design → Planning → Implementation → Ship
```

---

## Stage 1: Vision → Prototype

**Owner:** Product / Founder / Engineering lead
**Goal:** Validate the value proposition quickly with something people can try.

### Typical Tools

| Tool Category | Purpose |
|---------------|---------|
| UI prototyping tool | Generate an interface from natural-language requirements |
| App sandbox or full-stack prototyper | Stand up an interactive prototype quickly |
| Source control hosting | Save iterations early and preserve the source of truth |

### Process

1. Write a short problem statement.
2. Build a rough prototype as quickly as possible.
3. Export the code to version control early.
4. Get the prototype in front of users.
5. Iterate based on real feedback.

### Best Practices

- Save prototype iterations with clear notes.
- Prefer working flows over polished visuals early.
- Capture the actual states and transitions the prototype supports.
- Use feedback from real sessions to shape the next iteration.

---

## Stage 2: Prototype → Requirements / PRD

**Owner:** Product  
**Goal:** Turn working behavior into structured requirements.

### Typical Tools

| Tool Category | Purpose |
|---------------|---------|
| Documentation workspace | Store specs and collaborate on them |
| Repository connector | Let the model inspect the prototype code directly |
| Optional docs connector | Let the model write or update requirements in your docs system |

### Key Insight

Feed the working code to an LLM to extract a more complete PRD than you would usually get from memory alone.

### Requirements Generation Prompt

```markdown
Analyze this codebase and generate a product requirements document that captures:

## Required Sections

1. User-facing states and interactions
2. Data models and relationships
3. Error, loading, and empty states
4. Authentication and authorization rules
5. User stories derived from real behavior
6. Testable acceptance criteria derived from the implementation

## Output Format
- Markdown suitable for a shared documentation workspace
- Mermaid diagrams where useful
- Tables for important data models and flows
```

### Example Generic MCP Configuration

```json
{
  "mcpServers": {
    "docs": {
      "command": "your-docs-connector-command"
    },
    "repo": {
      "command": "your-repo-connector-command"
    }
  }
}
```

---

## Stage 3: Requirements → Tickets

**Owner:** Product / Engineering  
**Goal:** Break the requirements into implementable work items.

### Typical Tools

| Tool Category | Purpose |
|---------------|---------|
| Issue tracker | Store execution-ready work items |
| Issue tracker connector | Let the model create and update tickets |

### Ticket Generation Prompt

```markdown
You have access to:
- The requirements document in the docs workspace
- The codebase

For each feature:

1. Create or update an epic/project grouping
2. Break work into small, independently shippable issues
3. Add acceptance criteria and technical notes
4. Record dependencies and sequencing
5. Add labels such as frontend, backend, infra, or design

Prefer issues that can be completed in a few hours rather than many days.
```

### Best Practices

- Keep issues small and testable.
- Sequence backend/platform work before dependent frontend work.
- Link every ticket back to the underlying requirement.

---

## Stage 4: Design Validation

**Owner:** Design  
**Goal:** Validate that designs cover all required states and interactions.

### Typical Tools

| Tool Category | Purpose |
|---------------|---------|
| Design tool | Produce final mocks and flows |
| Design connector | Let the model inspect design frames and components |
| Shared review workspace | Capture gaps and decisions |

### Design Validation Prompt

```markdown
You have access to:
- The design file
- The requirements document
- The issue tracker

For each screen:

1. Map it to the relevant user story
2. Confirm loading, empty, error, and success states exist
3. Check responsive behavior and edge cases
4. Identify missing interactions or ambiguous flows
5. Update related tickets with design references and gaps
```

### Best Practices

- Validate the full state space, not just the happy path.
- Make design links discoverable from the corresponding tickets.
- Treat missing states as implementation blockers, not polish items.

---

## Stage 5: Technical Planning

**Owner:** Engineering leads
**Goal:** Decide implementation sequence, architecture fit, and constraints.

### Typical Tools

| Tool Category | Purpose |
|---------------|---------|
| Local coding assistant | Planning and codebase analysis |
| Autonomous coding agent | Multi-file implementation support |
| Repository guidance file | Constrain implementation patterns |

### The Repository Guidance Concept

Repository guidance files such as `AGENTS.md` help keep AI assistance consistent by documenting architectural patterns, testing expectations, naming conventions, and non-goals.

### Example `AGENTS.md` Topics

- Tech stack and approved infrastructure choices
- Backend and frontend architecture patterns
- API conventions
- Testing requirements
- Code review expectations
- Explicit anti-patterns and non-goals

### Technical Planning Prompt

```markdown
You have access to:
- The codebase
- The requirements document
- The design file
- The issue tracker
- AGENTS.md in the repository

For the selected workstream:

1. Analyze requirements and implicit constraints
2. Map the work to existing code patterns
3. Identify infrastructure, schema, and API changes
4. Sequence implementation safely
5. Call out review triggers such as security, performance, or breaking changes
```

---

## Stage 6: Autonomous Implementation

**Owner:** Engineering
**Goal:** Ship code with strong verification and targeted human oversight.

### Typical Tools

| Tool Category | Purpose |
|---------------|---------|
| Coding assistant or agent | Implement changes with tests |
| CI/CD pipeline | Run automated verification |
| PR review automation | Surface review findings quickly |

### Execution Loop

```text
For each ticket:
  1. Gather context from requirements, design, and AGENTS.md
  2. Create a focused branch
  3. Write or update tests first when practical
  4. Implement the change
  5. Run checks and fix failures
  6. Open a focused PR with context
  7. Address review feedback
  8. Merge once verified
```

### Implementation Prompt

```markdown
Implement the selected ticket.

## Context Gathering
1. Read the ticket and acceptance criteria
2. Review the relevant requirements sections
3. Review the design states and interactions
4. Read AGENTS.md for patterns and constraints

## Implementation Steps
1. Create a focused branch
2. Add or update tests for acceptance criteria
3. Implement the feature using existing patterns
4. Run verification checks
5. Commit with a clear conventional-style message

## PR Checklist
- [ ] Acceptance criteria covered
- [ ] Tests updated or added
- [ ] Build/lint/test checks passed
- [ ] Change is focused and low-risk
- [ ] Follows AGENTS.md guidance
```

---

## Example Generic MCP Configuration

```json
{
  "mcpServers": {
    "docs": {
      "command": "your-docs-connector-command"
    },
    "repo": {
      "command": "your-repo-connector-command"
    },
    "tracker": {
      "command": "your-issue-tracker-connector-command"
    },
    "design": {
      "command": "your-design-connector-command"
    },
    "database": {
      "command": "your-database-connector-command"
    },
    "filesystem": {
      "command": "your-filesystem-connector-command"
    }
  }
}
```

---

## Critical Success Factors

| Factor | Why It Matters |
|--------|----------------|
| Clear repository guidance | Keeps assistants aligned with your patterns |
| Atomic tickets | Makes autonomous execution safer |
| Strong tests | Lets assistants verify their own work |
| Type safety | Creates fast feedback loops |
| Good documentation | Reduces ambiguity and repeated mistakes |
| CI/CD quality gates | Prevents broken changes from shipping |

---

## Team Role Evolution

AI-assisted development shifts work from manual repetition toward direction, review, and systems thinking.

| Role | Traditional Focus | AI-Augmented Focus |
|------|-------------------|--------------------|
| Product | Write specs manually | Drive clearer requirements and decision logs |
| Design | Produce mockups only | Produce state-complete, validation-friendly designs |
| Tech Lead | Write critical paths personally | Define guidance, review risks, shape architecture |
| Engineer | Implement everything directly | Direct, verify, and refine assisted implementation |
| QA | Manual regression | Expand coverage, validate edge cases, improve testability |

### Mindset Shifts

1. From writing everything to reviewing and directing.
2. From implicit knowledge to explicit guidance.
3. From large batches to small verified increments.
4. From one-path execution to parallel investigation and synthesis.

---

## Getting Started Checklist

- [ ] Put clear guidance in `AGENTS.md`
- [ ] Choose connectors for docs, repo, tracker, and design if needed
- [ ] Break work into small independently verifiable tickets
- [ ] Set up CI/CD quality gates
- [ ] Test the prototype → requirements → tickets flow on one feature
- [ ] Iterate on prompts and preserve what works

---

## Resources

- [Model Context Protocol](https://modelcontextprotocol.io/)

---

*Last updated: May 2026*
