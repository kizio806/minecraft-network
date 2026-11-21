# Plugin Management Guide

Minecraft Network uses an automated, declarative plugin management engine implemented in Go.

---

## 📦 Declarative Plugin Bundles

Plugin dependencies are specified declaratively in YAML configuration manifests:

```yaml
plugins:
  - id: luckperms
    provider: modrinth
    version: "5.4.102"
    checksum: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  - id: placeholderapi
    provider: spiget
    resource_id: 6245
    version: "2.11.6"
```

---

## 🔒 Security & Verification Guarantees

1. **Checksum Enforcement:** Downloaded `.jar` files are validated against pinned SHA-256 hashes to prevent upstream supply-chain tampering.
2. **Provider Failover:** Graceful fallback across primary CDN endpoints and backup mirrors.
3. **Automated Audit:** CI scans plugin manifests for outdated versions and known security advisories.
