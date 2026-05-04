# Lessons Learned

Review this file at the start of each session and apply any relevant rules before changing code or documentation.

## How to use this file
- Add a new entry after any correction from the user.
- Capture the context, the correction, and the rule that should prevent the same mistake.
- Prefer specific prevention rules over vague reminders.

## Entry Template

```markdown
### YYYY-MM-DD — Short lesson title
- Context:
- Correction from user:
- Rule to follow next time:
```

## Lessons

### 2026-05-03 — Use subagents for parallel investigation
- Context: workflow guidance and orchestration expectations for non-trivial work.
- Correction from user: make subagent usage explicit when work benefits from research, exploration, or parallel analysis.
- Rule to follow next time: proactively kick off focused subagents for bounded research/exploration tasks instead of keeping all investigation in the main context.

### 2026-05-04 — Use the standard AGENTS filename
- Context: repository-scoped instruction files for generic LLM/agent tooling.
- Correction from user: rename `AGENT.md` to `AGENTS.md` rather than removing generic agent guidance.
- Rule to follow next time: use `AGENTS.md` as the canonical repository instruction filename unless the user explicitly asks for a different convention.

### 2026-05-04 — Remove vendor-specific instruction files, not generic guidance
- Context: repository cleanup for LLM-agnostic tooling.
- Correction from user: remove vendor-specific instruction files while keeping generic repository guidance like `AGENTS.md`.
- Rule to follow next time: when asked to make the repo tool-agnostic, delete vendor-specific instruction files but preserve generic guidance unless the user explicitly asks to remove that too.
