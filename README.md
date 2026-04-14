# Token Heatmap

Turn local coding agent usage into a GitHub-style token heatmap.

`Token Heatmap` scans local usage data from `Codex`, `Claude Code`, and `OpenCode`, stores it in SQLite, generates a 365-day heatmap, and can sync the result to GitHub.

## Supported Sources

- `Codex`: `~/.codex/sessions`, `~/.codex/archived_sessions`
- `Claude Code`: `~/.config/claude/projects`, `~/.claude/projects`
- `OpenCode`: `~/.local/share/opencode/opencode.db`

## Core Commands

```bash
./tokenheat collect
./tokenheat report today
./tokenheat report today --json
./tokenheat generate heatmap
./tokenheat run daily --profile-repo-dir ../849261680
./tokenheat schedule install --profile-repo-dir ../849261680
```

## What It Does

- Collects local token usage into `~/.tokenheat/tokenheat.db`
- Generates:
  - `docs/usage.json`
  - `docs/heatmap.svg`
- Syncs the heatmap to the project repo and an optional GitHub profile repo
- Supports daily macOS automation via `launchd`

## Menu Bar App

- macOS menu bar shell lives in `apps/macos/TokenHeatMenu`
- Build with:

```bash
./scripts/build-tokenheat-menu.sh
```

- Requires full Xcode
- Produces: `dist/Token Heatmap.app`
