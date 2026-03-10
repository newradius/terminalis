# Remote-Enhanced Autosuggestions Design

Extend existing local-history autosuggestions with remote shell history and remote command name completion.

## Data Sources (priority order)

1. **Local history** (existing) — full command-line matches, highest priority
2. **Remote shell history** — read via SFTP on connect. Full command-line matches.
3. **Remote available commands** — fetched via SSH exec channel. First-word (command name) only.

## Backend

New Go method `FetchRemoteCompletions(tabID string)` returns `{history: []string, commands: []string}`:
- Opens separate SSH exec channel to detect shell via `echo $SHELL`
- Reads appropriate history file via SFTP (~/.bash_history, ~/.zsh_history, ~/.local/share/fish/fish_history)
- Runs `compgen -c` (bash) or `whence -pm '*'` (zsh) for command names, falls back to `ls /usr/bin /usr/sbin /bin /sbin`
- Cap remote history at 5000 entries

## Frontend Matching Logic

- If input contains a space (past first word): search local history, then remote history (full line prefix match)
- If first word only: search local history, then remote history, then remote commands (prefix match)

## Fetch Timing

- On SSH connect (background, non-blocking), after `ssh:connected` event
- Stored in memory per-tab, not persisted
- Local terminals skip remote fetch

## Edge Cases

- SFTP/history file unavailable: silently skip
- compgen unavailable: fall back to ls /usr/bin etc.
- Fish history: parse `- cmd:` lines
