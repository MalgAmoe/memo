# memo

Persistent memory system for Claude Code. Store and retrieve knowledge across sessions.

## Setup

```bash
# Start Redis and embeddings service
docker compose up -d

# Build
make build

# Install to PATH
make install
```

## Usage

```bash
# Recover context at session start
memo context

# Search memories
memo similar "how does X work" --here

# Store something
memo remember learned "API returns JSON, not XML"
memo remember preference "User prefers tabs over spaces"
memo remember fact "Config lives in ~/.config/app"

# Other commands
memo list --here              # List this project's memories
memo projects                 # Show all projects
memo get <id>                 # View specific memory
memo update <id> "new text"   # Edit memory
memo related <id>             # Find similar memories
memo forget <id>              # Delete memory
```

## Types

- `fact` - Objective information
- `learned` - Discoveries, gotchas, how things work
- `preference` - User preferences, workflow choices
- `context` - Project background

## Scoping

Memories are auto-tagged with the current project (git repo or directory name).

- `memo context` shows only current project
- `memo similar "query" --here` searches current project only
- `memo similar "query"` searches everything (including global)
- Memories without a project tag are global

## Architecture

```
memo (Go CLI)
    |
    +-- Redis 8 (JSON documents + RediSearch + Vector Sets)
    |
    +-- text-embeddings-inference (local nomic-embed-text-v1.5)
```

## For Claude Code

Add to `~/.claude/CLAUDE.md`:

```markdown
## Memory

Start sessions with `memo context`. Before answering questions about this project, try `memo similar "topic" --here`.

Save important discoveries:
- `memo remember learned "..."` - gotchas, patterns, how things work
- `memo remember preference "..."` - user choices, workflow preferences
- `memo remember fact "..."` - config locations, API details, decisions made
```
