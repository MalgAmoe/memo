# Claude Memory System

You have access to `memo` - a persistent memory system. Use it to store and retrieve knowledge across sessions.

## Quick Start

```bash
# At session start - recover context for this project
memo context

# Search semantically
memo similar "what you're looking for" --here

# Remember something important
memo remember <type> "content" --tags tag1,tag2
```

## Commands

| Command | Description |
|---------|-------------|
| `memo context [limit]` | Show project brief + memories |
| `memo similar <query> [--here] [--limit N]` | Semantic search |
| `memo recall <query>` | Full-text search |
| `memo remember <type> <content> [--tags] [--force]` | Store a memory (dedup checked; --force to skip) |
| `memo get <id>` | View a specific memory |
| `memo update <id> <content>` | Edit a memory's content |
| `memo tag <id> <tag>` | Add a tag to a memory |
| `memo related <id>` | Find similar memories |
| `memo forget <id>` | Delete a memory |
| `memo merge <id1> <id2> ["content"]` | Merge two memories (optional content override) |
| `memo brief [--refresh]` | Show/regenerate project understanding (LLM-synthesized) |
| `memo dedup [--project P]` | Find redundant/outdated memories (LLM-powered) |
| `memo list [--type T] [--project P] [--here]` | List with filters |
| `memo prune [--days N] [--delete]` | Find stale memories (dry run by default) |
| `memo stats` | Show counts by type |
| `memo projects` | List all projects with memory counts |

## Types

- `fact` - Objective information
- `learned` - Something discovered/figured out
- `preference` - User preferences
- `context` - Project background

## Scoping

- **Project-scoped**: Memories are auto-tagged with `project:<name>` based on git repo or directory
- **Global**: Memories without a project tag are global (appear everywhere)
- Use `--here` to filter to current project only

## When to Use

- **Session start**: Run `memo context` to recover project knowledge (shows brief + individual memories)
- **Before answering**: Check `memo similar "topic" --here` if unsure about project details
- **Learning something**: `memo remember learned "..."` - save proactively, don't wait to be asked
- **User preference**: `memo remember preference "..."` (consider keeping global - no project tag)
- **Before context compaction**: Save important discoveries
- **Periodic cleanup**: Run `memo dedup` to find redundant/outdated memories, then decide what to merge/update/forget

## How to Write Good Memories

- **One fact per memory** - never mix topics. "Redis uses port 6379" not "Redis uses port 6379 and the API key is in .env"
- **Absolute dates only** - never "today" or "recently". Use "2026-03-13" so the memory makes sense months later
- **Preserve specifics** - version numbers, port numbers, env var names, file paths. These lose value if generalized
- **When uncertain, remember it** - dedup catches redundancy. Missing knowledge is worse than duplicate knowledge
- **Use update over create** - if a memory exists on the same topic, `memo update <id>` rather than creating a new one. `remember` shows related memories to help you decide
- **Prefix with topic when useful** - "Billing: seconds deducted on job completion" makes it easier to find and update later
