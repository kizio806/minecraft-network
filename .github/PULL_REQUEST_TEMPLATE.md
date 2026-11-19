## Description
<!-- Provide a concise summary of the changes and motivation for this pull request. -->

## Type of Change
- [ ] Bug fix (non-breaking change fixing an issue)
- [ ] New feature (non-breaking change adding functionality)
- [ ] Breaking change (fix or feature changing existing behavior)
- [ ] Documentation update
- [ ] Refactoring / tooling enhancement

## Architectural Compliance
- [ ] **Port Isolation:** No backend ports or PostgreSQL ports are published to host.
- [ ] **Image Pinned:** All container images use `@sha256:` immutable digests.
- [ ] **Supply Chain:** No binary `.jar` or `.zip` files added to git (all declared via manifest/lockfiles).
- [ ] **Zero Secrets:** No plaintext passwords or tokens committed.
- [ ] **Zero Redis:** No Redis dependencies introduced.

## Verification Checklist
- [ ] `./bin/network doctor` passes
- [ ] `./bin/network plugins validate` passes
- [ ] `go test -race ./...` passes
- [ ] `docker compose -f docker-compose.yml config --quiet` passes
- [ ] All commits follow Conventional Commits standard
