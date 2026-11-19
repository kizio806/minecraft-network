# Rule: Multi-Agent Coordination

## Purpose
Prevent concurrency collisions, race conditions, and context contamination when multiple AI agents work concurrently on the repository.

## Scope
Applies to multi-agent interactions across Claude Code, Cursor, GitHub Copilot, OpenAI Codex, and Antigravity.

## MUST
- **MUST** read relevant domain documentation (`.agents/context/`, `.agents/rules/`) before initiating code modifications.
- **MUST** isolate edits to atomic file blocks and verify that no concurrent lock (`.network-operation.lock`) is active before running CLI operations.
- **MUST** respect the role boundaries defined in `.agents/roles/` (e.g. Security Auditor does not edit menus; Plugin Specialist does not reconfigure Docker networks).
- **MUST** verify the complete quality gauntlet (`doctor`, `health`, unit tests with `-race`) before concluding any turn.

## MUST NOT
- **MUST NOT** perform simultaneous modifications to shared configuration locks (`configs/plugins/lock.yml`) without coordination.
- **MUST NOT** bypass git status checks or overwrite files without inspecting diffs.
- **MUST NOT** leave uncommitted scratch files, mock scripts, or temporary test data in the repository root.

## Verification
Inspect git working tree before and after agent tasks:
```bash
git status -s
```
Output must contain only authorized, tracked modifications.
