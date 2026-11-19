---
name: audit-codebase
description: Audit repository state, detect orphaned files, check image digest pinning, and ensure configuration integrity.
---

# Skill: Codebase Audit

## Use Case
Execute this skill when initiating architectural reviews, onboarding new agents, or verifying repository hygiene before and after major modifications.

## Inputs & Prerequisites
- Linux shell with `git`, `docker`, and `find` available in PATH.
- Read access to repository root and configuration trees.

## Procedure

### 1. Detect Untracked Runtime & Scratch Files
```bash
git status -s
```
Verify that no dynamic Minecraft runtime files (`session.lock`, `uid.dat`, `level.dat_old`, `playerdata/`) or unencrypted secret keys are present in Git.

### 2. Verify Docker Image Pinning
Check `docker-compose.yml` for any unpinned images:
```bash
grep -n "image:" docker-compose.yml
```
Every service image MUST include `@sha256:` immutable digest hashes.

### 3. Verify Plugin Manifest & Lockfile Alignment
```bash
./bin/network plugins validate
```

### 4. Audit File Permissions on Secrets
```bash
stat -c "%a %n" secrets/ secrets/*
```
Directory must be `0700` and files must be `0600`.

## Validation
- Git status output reports clean working directory or only authorized modifications.
- Zero unpinned images found in `docker-compose.yml`.
- Plugin validation exits with code 0.
- All secrets verify with mode `600`.

## Limitations & Edge Cases
- Does not modify files automatically; only detects and reports discrepancies.
- If uncommitted files exist, do not use `git clean` without human confirmation.
