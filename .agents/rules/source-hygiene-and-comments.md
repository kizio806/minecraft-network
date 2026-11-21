# Source Hygiene, Comment Policy, and Quality Standards

## Purpose
In the `minecraft-network` codebase, all Go code and configuration files must maintain high readability, clean architecture, and strict source hygiene.

---

## 1. Commenting Philosophy ("Clean Code" Standard)
- **Self-Documenting Code:** Function names, variable names, and package boundaries must be expressive and self-explanatory.
- **Explain "WHY", Not "WHAT":** Inline comments (`//`) are reserved strictly for non-obvious context, architectural trade-offs, network packet constraints, or path traversal safeguards.
- **No Redundant Comments:** Never write comments that merely rephrase the function or variable name directly underneath.
- **English Exclusively:** All comments, GoDoc, identifiers, error messages, and log strings must be written in English.

---

## 2. API Documentation (GoDoc)
- **Exported Identifiers:** Every exported type, function, method, and constant must have clear GoDoc comments adhering to Go standard conventions (`// TypeName does ...`).
- **Internal / Private Helpers:** Keep unexported helper functions concise and self-documenting; formal GoDoc is not required unless clarifying non-trivial internal logic.

---

## 3. Strict Hygiene Invariants
- **Zero Markers:** Never commit `TODO`, `FIXME`, or `XXX` tags in Go source files. Outstanding work must be tracked via GitHub Issues.
- **Zero Unhandled Errors:** Never silently drop errors (`_ = err`) without explicit, justified architectural reasoning.
- **No Path Traversal:** File I/O operations must validate paths with `filepath.Clean`.
