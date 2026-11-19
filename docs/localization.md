# Localization & Language Bundles (i18n)

Minecraft Network includes a centralized localization framework enabling rapid language switching across all proxy, lobby, and survival templates.

---

## 1. Selecting the Active Language

Set the active locale in `configs/network.yml`:

```yaml
# Active language: "en" (English) or "pl" (Polish)
locale: "en"
```

Then execute:
```bash
./bin/network install
```

The CLI compiler loads the corresponding locale bundle from `locales/<locale>.yml`. If a message key is missing in the chosen language, it automatically falls back to `locales/en.yml`.

---

## 2. Directory Layout & Adding a Language

All locale dictionaries reside under `locales/`:

```text
locales/
  ├── en.yml   # Base language bundle (authoritative fallback)
  ├── pl.yml   # Polish language bundle
  └── de.yml   # Example: add new languages simply by creating <lang>.yml
```

### Adding a New Translation
1. Copy `locales/en.yml` to `locales/<lang>.yml` (e.g. `locales/de.yml`).
2. Translate the values under the `messages:` key.
3. Update `locale: "<lang>"` in `configs/network.yml` and run `./bin/network install`.
