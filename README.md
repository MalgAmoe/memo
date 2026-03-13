# memo

Persistent memory system for Claude Code. Stores facts, synthesizes understanding, keeps knowledge clean across sessions.

## Setup

```bash
# Start Redis and embeddings service
docker compose up -d

# Add your Fireworks API key (for LLM features: brief, dedup)
echo "FIREWORKS_API_KEY=fw_xxx" > .env

# Build (bakes API key into binary) and install to PATH
make install

# Initialize search index
memo init
```

## Usage

```bash
# Recover context at session start (shows project brief + memories)
memo context

# Search memories
memo similar "how does X work" --here
memo recall "keyword"

# Store something
memo remember learned "API returns JSON, not XML"
memo remember fact "Config lives in ~/.config/app"
memo remember preference "User prefers tabs over spaces"

# Project understanding (LLM-synthesized brief)
memo brief                    # Show current brief
memo brief --refresh          # Regenerate

# Cleanup (LLM-powered)
memo dedup                    # Find redundant/outdated memories
memo dedup --project foo      # For a specific project

# Other commands
memo list --here              # List this project's memories
memo get <id>                 # View specific memory
memo update <id> "new text"   # Edit memory
memo related <id>             # Find similar memories
memo merge <id1> <id2> "text" # Merge two memories
memo forget <id>              # Delete memory
memo prune [--days N]         # Find stale memories
memo stats                    # Memory counts by type
memo projects                 # Show all projects
```

## Types

- `fact` - Objective information
- `learned` - Discoveries, gotchas, how things work
- `preference` - User preferences, workflow choices
- `context` - Project background

## Scoping

Memories are auto-tagged with the current project (git repo or directory name).

- `memo context` shows only current project (with synthesized brief)
- `memo similar "query" --here` searches current project only
- `memo similar "query"` searches everything
- Memories without a project tag are global

## Architecture

```
memo (Go CLI)
    |
    +-- Redis 8 (JSON documents + RediSearch + Vector Sets)
    |
    +-- text-embeddings-inference (local nomic-embed-text-v1.5)
    |
    +-- Fireworks API / Kimi K2.5 (brief synthesis, dedup analysis)
```

## Writing Good Memories

- One fact per memory — never mix topics
- Use absolute dates, not "today" or "recently"
- Preserve specifics — version numbers, ports, env var names
- Prefer `memo update <id>` over creating duplicates
- When uncertain, remember it — dedup catches redundancy
