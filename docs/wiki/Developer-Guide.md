# Developer & Contributor Guide

This guide details how to develop, test, and contribute to the **Minecraft Network** orchestration platform.

---

## 🛠️ CLI Development (Go)

The orchestration CLI is implemented in Go 1.22+.

### Build & Test:
```bash
# Run unit tests across all packages
go test ./...

# Build the network management binary
go build -o bin/network-cli ./cmd/network-cli
```

---

## 📜 Coding & Quality Standards

- Adhere to idiomatic Go naming and formatting (`go fmt` / `golangci-lint`).
- Maintain zero compiler warnings and cleanly passing CI workflows.
- Update documentation in `docs/` and `docs/wiki/` when changing CLI commands or deployment manifests.
