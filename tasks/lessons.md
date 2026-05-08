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

### 2026-05-05 — Be explicit about whether code changed
- Context: answering architecture questions mid-implementation.
- Correction from user: make it clear whether I actually changed the code or was only giving guidance.
- Rule to follow next time: explicitly state when no repo changes were made, especially after a design-only or clarification response.

### 2026-05-06 — Keep OAuth consent human-readable and cross-site safe
- Context: self-hosted OAuth consent flow for ChatGPT MCP.
- Correction from user: the consent page exposed raw client metadata and the Approve action did not complete the redirect flow.
- Rule to follow next time: for third-party OAuth consent screens, show the app name plus human-readable requested access only, and treat approval submits as cross-site/browser-embedded flows by using secure cookie settings and POST-safe redirects.

### 2026-05-06 — Verify deployed OAuth/browser flows at the HTTP boundary
- Context: production ChatGPT MCP auth still failed after the consent-flow fix was merged and deployed.
- Correction from user: the same blank-popup behavior persisted in production even after the previous fix shipped.
- Rule to follow next time: when debugging deployed OAuth/browser issues, verify the live HTTP behavior (headers, preflight handling, redirects) against production before assuming the previous hypothesis fully solved the problem.

### 2026-05-06 — Don’t block first-party OAuth form posts with third-party origin rules
- Context: tightening OAuth endpoint CORS/origin checks for ChatGPT compatibility.
- Correction from user: after the stricter origin gate shipped, clicking Approve redirected back to FlyingForge instead of completing OAuth.
- Rule to follow next time: when adding origin restrictions to OAuth endpoints, explicitly allow the app’s own public origin/issuer for first-party browser form submissions in addition to third-party client origins.

### 2026-05-06 — When authorize succeeds but no token exchange follows, inspect popup response mode
- Context: ChatGPT connector OAuth approval kept opening a blank popup even after the consent and CORS fixes were deployed.
- Correction from user: the browser still showed the same spinner-and-blank-popup behavior, so the prior fixes had not resolved the final handoff.
- Rule to follow next time: if production logs show `/oauth/authorize` succeeding but no `/oauth/token` request ever arrives, verify whether the client expects popup-oriented `response_mode=web_message` handling instead of a normal redirect callback.

### 2026-05-06 — Confirm the live request parameters before patching a specific OAuth branch
- Context: I added popup `response_mode` support, but production still showed the same spinner/blank-popup behavior.
- Correction from user: the new patch still did not work in the real ChatGPT flow.
- Rule to follow next time: before committing to a response-mode-specific OAuth fix, inspect the live authorize logs to confirm whether the client is actually sending that parameter and patch the active branch of the flow first.

### 2026-05-07 — Read the browser CSP error literally in OAuth redirect bugs
- Context: the ChatGPT approval flow still failed after redirect-path fixes.
- Correction from user: the browser console showed a precise CSP `form-action` violation for the ChatGPT callback URL.
- Rule to follow next time: when a browser surfaces a CSP violation during OAuth approval, patch the exact blocked directive first—especially `form-action` on consent pages that POST and then redirect to a third-party callback.
