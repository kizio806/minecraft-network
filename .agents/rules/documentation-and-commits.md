# Rule: Documentation & Commits

## Purpose
Maintain strict documentation accuracy, ensure single source of truth across architecture and roadmap documents, and enforce Conventional Commits.

## Scope
Applies to all Markdown documentation files (`AGENTS.md`, `.agents/`, `docs/`, `PROJECT.md`, `ROADMAP.md`), README files, and Git commit messages.

## MUST
- **MUST** format all commit messages according to the Conventional Commits specification:
  ```
  <type>(<scope>): <short description in imperative mood>
  ```
  Valid types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`.
- **MUST** keep `PROJECT.md` and `ROADMAP.md` updated as the single source of truth whenever system behavior or task status changes.
- **MUST** write clean, structured Markdown using standard headings (`#`, `##`, `###`), tables, and code fences.
- **MUST** preserve existing documentation integrity, licenses, and comments in configuration files.

## MUST NOT
- **MUST NOT** write generic commit messages (e.g. `update`, `fix`, `changes`, `wip`).
- **MUST NOT** leave orphaned, conflicting, or duplicated documentation files across the repository.
- **MUST NOT** document hypothetical or unimplemented features as completed.

## Verification
Review git diff and check commit message syntax:
```bash
git diff --staged
git status -s
```
Must show only intended changes with clean working directory.
