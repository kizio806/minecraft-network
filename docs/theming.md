# Theming Architecture & Color Presets

Minecraft Network features a centralized, token-based color theming system. Instead of hardcoding hex values across dozens of plugin configurations, colors are declared as semantic tokens and resolved during runtime rendering (`./bin/network install`).

---

## 1. Theme Configuration

In `configs/network.yml`, define the active preset:

```yaml
theme:
  preset: "amber-sunset" # Selected preset
  presets:
    amber-sunset:
      primary: "#f6d365"
      secondary: "#fda085"
      accent: "#ff7e5f"
      danger: "#c9453e"
      success: "#6fc93e"
      gold: "#FFD700"
    cyber-blue:
      primary: "#00d2ff"
      secondary: "#3a7bd5"
      accent: "#00f2fe"
      danger: "#ff416c"
      success: "#00b09b"
      gold: "#f9d423"
```

---

## 2. Standard Semantic Tokens

| Token | Semantic Purpose | Example Usage |
|---|---|---|
| `__THEME_PRIMARY__` | Brand primary color | Title highlights, compass item name, TAB prefixes |
| `__THEME_SECONDARY__` | Secondary brand color | Gradients, server name secondary, menu text |
| `__THEME_ACCENT__` | Visual accent color | Highlighted options, arrows (`»`), separators |
| `__THEME_DANGER__` | Critical warnings & combat | Anti-cheat alerts, PvP tag, punishment notices |
| `__THEME_SUCCESS__` | Positive actions | Online status dots, toggles enabled, safe zones |
| `__THEME_GOLD__` | Economy & elite status | SVIP rank badges, currency rewards |

---

## 3. Formatting Syntaxes Supported

- **MiniMessage / Adventure:** `<__THEME_PRIMARY__>` or `<gradient:__THEME_PRIMARY__:__THEME_SECONDARY__>`
- **Legacy Section / Ampersand:** `&__THEME_PRIMARY__` (e.g. `&#f6d365` when primary is `#f6d365`)
