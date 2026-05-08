# Repository Agent Workflow

This repository expects AI coding assistants to follow the workflow below for any non-trivial task.

## 1. Plan First
- Enter plan mode for any non-trivial task (3+ steps, architectural decisions, or meaningful verification work).
- If your client does not support a dedicated plan mode, write the plan explicitly before editing code.
- Use plan mode for verification steps, not just implementation steps.
- Write detailed specs and checklists up front to reduce ambiguity.
- If something goes sideways, stop and re-plan immediately instead of pushing through a bad plan.

## 2. Stay on the Primary Model
- Default to the primary GPT model for research, planning, implementation, and review.
- Do not offload work to local agents or subagents unless the user explicitly asks for delegation.
- Keep the task focused with good planning and scoped verification instead of automatic delegation.
- If delegation is ever used, treat it as an exception and explain why it is necessary.

## 3. Self-Improvement Loop
- Review `tasks/lessons.md` at the start of each session for relevant lessons.
- After any correction from the user, update `tasks/lessons.md`.
- Record concrete rules that prevent repeating the same mistake.
- Iterate ruthlessly until repeated mistakes stop recurring.

## 4. Verification Before Done
- Never mark a task complete without proving it works.
- When relevant, compare baseline behavior against changed behavior.
- Always run the relevant tests, check logs or errors, and demonstrate correctness.
- Ask: *Would a staff engineer approve this?*
- If verification fails, stop and re-plan instead of hand-waving the result.

## 5. Demand Elegance (Balanced)
- For non-trivial changes, ask whether there is a more elegant solution.
- If a fix feels hacky, step back and ask: *Knowing what I know now, what's the correct solution?*
- Skip this for simple fixes to avoid over-engineering.
- Prefer root-cause fixes over temporary patches.

## 6. Autonomous Bug Fixing
- When given a bug report, own the investigation and fix without hand-holding.
- Use logs, errors, reproductions, and failing tests.
- Avoid unnecessary context switching for the user.
- Proactively address failing CI that is relevant to the task.

## Task Management
- Plan first by writing checkable items in `tasks/todo.md`.
- `tasks/todo.md` is a local working file and must remain untracked in Git.
- Confirm the plan before starting non-trivial or high-risk implementation. For tiny, low-risk changes with clear intent, the written plan is enough.
- Mark items complete as work progresses.
- Add a review section to `tasks/todo.md` with verification notes, risks, and follow-ups.
- Use `tasks/lessons.md` as the tracked repository memory for user corrections and recurring mistakes.

### Suggested `tasks/todo.md` Template

```markdown
# Task Plan

## Task
- Short description of the work

## Plan
- [ ] Step 1
- [ ] Step 2
- [ ] Step 3

## Progress Notes
- Key updates, decisions, and blockers

## Review
- Verification:
- Risks:
- Follow-ups:
```

### Suggested `tasks/lessons.md` Entry Format

```markdown
### YYYY-MM-DD — Short lesson title
- Context:
- Correction from user:
- Rule to follow next time:
```
