# Contributing to Minecraft Network

Thank you for your interest in contributing to **Minecraft Network**! This document outlines our development process, guidelines, and quality gates for external contributions.

---

## 1. Architectural Guardrails (5 Golden Invariants)

All pull requests must respect the project's core architectural constraints:

1. **Port Isolation:** Only `proxy` (`0.0.0.0:25565:25565/tcp`) may publish ports to the public network. Game backends (`auth`, `lobby`, `survival`) and `postgres` must NEVER bind to host interfaces.
2. **Immutable Image Digests:** Every container image in `docker-compose.yml`, `.env.example`, and CI must be pinned to an immutable `@sha256:` digest. Mutable tags (`latest`, `alpine`) are prohibited.
3. **Declarative Plugin Supply Chain:** Never place JAR files or datapacks directly into Git. Plugins must be declared in `configs/plugins/manifest.yml` and pinned in `configs/plugins/lock.yml`. Datapacks and standalone components must be declared in `configs/plugins/components.yml` and pinned in `configs/plugins/components.lock.yml`.
4. **Secret Protection:** Passwords, HMAC tokens, and private keys reside in `./secrets/` with strict mode `0600` (directory `0700`). Never commit secrets or plaintext passwords.
5. **Database Consolidation & Zero Redis:** All persistent authentication and permissions reside in isolated schemas on PostgreSQL 16 Alpine. Redis is banned across all services.

---

## 2. Development Setup

### Requirements
- Linux (Ubuntu 22.04+ recommended) or macOS
- Docker Engine & Docker Compose Plugin (v2.27+)
- Go 1.24+ (or Docker for building and testing)

### Local Build & Test
```bash
# 1. Build the control CLI (host Go or automated containerized fallback)
make build

# 2. Run unit tests with race detection
make test-race

# 3. Run static analysis (golangci-lint)
make lint

# 4. Validate plugin supply chain
./bin/network plugins validate

# 5. Verify Compose configurations
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml --profile monitoring config --quiet
```

---

## 3. Pull Request Guidelines

1. **Branch Naming:** Use descriptive branch names: `feat/<feature-name>`, `fix/<bug-name>`, `refactor/<scope>`.
2. **Conventional Commits:** All commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat(scope): add new feature`
   - `fix(scope): fix bug`
   - `docs(scope): update documentation`
   - `refactor(scope): refactor code without behavioral change`
3. **Automated Quality Gate:** Every pull request must pass CI checks in `.github/workflows/ci.yml`:
   - Go unit tests with `-race` and Codecov coverage
   - Go vet and formatting (`gofmt`)
   - `golangci-lint` static analysis
   - CodeQL security analysis
   - SonarCloud quality gate
   - ShellCheck on all shell scripts
   - Yamllint on Compose and infrastructure configs
   - Gitleaks secret leak detection
