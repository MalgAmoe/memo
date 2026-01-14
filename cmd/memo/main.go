package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"memo/internal"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	client := internal.NewClient()
	defer client.Close()

	var err error
	switch cmd {
	case "init":
		err = cmdInit(client)
	case "remember":
		err = cmdRemember(client, args)
	case "recall":
		err = cmdRecall(client, args)
	case "similar":
		err = cmdSimilar(client, args)
	case "context":
		err = cmdContext(client, args)
	case "list":
		err = cmdList(client, args)
	case "get":
		err = cmdGet(client, args)
	case "forget":
		err = cmdForget(client, args)
	case "update":
		err = cmdUpdate(client, args)
	case "tag":
		err = cmdTag(client, args)
	case "related":
		err = cmdRelated(client, args)
	case "reindex":
		err = cmdReindex(client)
	case "stats":
		err = cmdStats(client)
	case "projects":
		err = cmdProjects(client)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdInit(c *internal.Client) error {
	fmt.Println("Initializing memo index...")
	if err := c.Init(); err != nil {
		return err
	}
	fmt.Println("Index created.")
	return nil
}

func cmdRemember(c *internal.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: memo remember <type> <content> [--tags t1,t2]")
	}

	memType := args[0]

	// Parse content and tags
	var contentParts []string
	var tags []string

	for i := 1; i < len(args); i++ {
		if args[i] == "--tags" && i+1 < len(args) {
			tags = strings.Split(args[i+1], ",")
			i++
		} else {
			contentParts = append(contentParts, args[i])
		}
	}

	content := strings.Join(contentParts, " ")
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	project := internal.GetProject()
	memo, err := c.Remember(memType, content, tags, project)
	if err != nil {
		return err
	}

	// Embed asynchronously (use document embedding for storage)
	go func() {
		embedding, err := internal.GetDocumentEmbedding(content)
		if err != nil {
			return
		}
		c.EmbedMemory(memo.ID, embedding)
	}()

	fmt.Printf("Remembered [%s]: %s\n", memo.ID, content)
	return nil
}

func cmdRecall(c *internal.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: memo recall <query> [limit]")
	}

	query := args[0]
	limit := 10
	if len(args) > 1 {
		if l, err := strconv.Atoi(args[1]); err == nil {
			limit = l
		}
	}

	memos, err := c.Recall(query, limit)
	if err != nil {
		return err
	}

	fmt.Printf("%d results found\n\n", len(memos))
	for _, m := range memos {
		fmt.Printf("[%s] (%s) %s\n", m.ID, m.Type, m.Content)
	}
	return nil
}

func cmdSimilar(c *internal.Client, args []string) error {
	var query string
	var project string
	limit := 5

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--here":
			project = internal.GetProject()
		case "--limit":
			if i+1 < len(args) {
				if l, err := strconv.Atoi(args[i+1]); err == nil {
					limit = l
				}
				i++
			}
		default:
			if query == "" {
				query = args[i]
			}
		}
	}

	if query == "" {
		return fmt.Errorf("usage: memo similar <query> [--here] [--limit N]")
	}

	if project != "" {
		fmt.Printf("Searching for: %s (project: %s)\n", query, project)
	} else {
		fmt.Printf("Searching for: %s\n", query)
	}

	embedding, err := internal.GetEmbedding(query)
	if err != nil {
		return err
	}

	results, err := c.Similar(embedding, limit, project)
	if err != nil {
		return err
	}

	fmt.Println()
	if len(results) == 0 {
		fmt.Println("No matching memories found.")
		return nil
	}

	for _, r := range results {
		fmt.Printf("[%s] (%s) (%s) %s\n", r.Memory.ID, r.Score, r.Memory.Type, r.Memory.Content)
	}
	return nil
}

func cmdContext(c *internal.Client, args []string) error {
	limit := 10
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil {
			limit = l
		}
	}

	project := internal.GetProject()
	fmt.Printf("Context for project: %s\n", project)
	fmt.Println("================================")
	fmt.Println()

	memos, err := c.Context(project, limit)
	if err != nil {
		return err
	}

	if len(memos) == 0 {
		fmt.Println("No memories found for this project.")
		fmt.Println()
		fmt.Println("Start remembering with:")
		fmt.Println("  memo remember fact \"something important\"")
		return nil
	}

	for _, m := range memos {
		fmt.Printf("[%s] (%s) %s\n", m.ID, m.Type, m.Content)
	}
	return nil
}

func cmdList(c *internal.Client, args []string) error {
	var typeFilter, tagFilter, projectFilter string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				typeFilter = args[i+1]
				i++
			}
		case "--tag":
			if i+1 < len(args) {
				tagFilter = args[i+1]
				i++
			}
		case "--project":
			if i+1 < len(args) {
				projectFilter = args[i+1]
				i++
			}
		case "--here":
			projectFilter = internal.GetProject()
		}
	}

	// If project filter, add it as tag filter
	if projectFilter != "" {
		if tagFilter != "" {
			tagFilter = tagFilter + "|project*"
		} else {
			tagFilter = "project*"
		}
	}

	memos, err := c.List(typeFilter, tagFilter, 100)
	if err != nil {
		return err
	}

	// Filter by project client-side and extract project for display
	var filtered []internal.Memory
	for _, m := range memos {
		proj := getProjectFromTags(m.Tags)
		if projectFilter != "" && proj != projectFilter {
			continue
		}
		filtered = append(filtered, m)
	}

	fmt.Printf("%d memories\n\n", len(filtered))
	for _, m := range filtered {
		proj := getProjectFromTags(m.Tags)
		fmt.Printf("[%s] (%s) [%s] %s\n", m.ID, m.Type, proj, m.Content)
	}
	return nil
}

func getProjectFromTags(tags []string) string {
	for _, tag := range tags {
		if len(tag) > 8 && tag[:8] == "project:" {
			return tag[8:]
		}
	}
	return "?"
}

func cmdGet(c *internal.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: memo get <id>")
	}

	memo, err := c.Get(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("ID:       %s\n", memo.ID)
	fmt.Printf("Type:     %s\n", memo.Type)
	fmt.Printf("Content:  %s\n", memo.Content)
	fmt.Printf("Tags:     %s\n", strings.Join(memo.Tags, ", "))
	fmt.Printf("Created:  %s\n", memo.Created)
	fmt.Printf("Accessed: %s\n", memo.Accessed)
	fmt.Printf("Access#:  %d\n", memo.AccessCount)
	return nil
}

func cmdForget(c *internal.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: memo forget <id>")
	}

	if err := c.Forget(args[0]); err != nil {
		return err
	}

	fmt.Printf("Forgot: %s\n", args[0])
	return nil
}

func cmdTag(c *internal.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: memo tag <id> <tag>")
	}

	id := args[0]
	tag := args[1]

	if err := c.AddTag(id, tag); err != nil {
		return err
	}

	fmt.Printf("Tagged [%s] with: %s\n", id, tag)
	return nil
}

func cmdUpdate(c *internal.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: memo update <id> <content>")
	}

	id := args[0]
	content := strings.Join(args[1:], " ")

	if err := c.Update(id, content); err != nil {
		return err
	}

	// Re-embed asynchronously (use document embedding for storage)
	go func() {
		embedding, err := internal.GetDocumentEmbedding(content)
		if err != nil {
			return
		}
		c.EmbedMemory(id, embedding)
	}()

	fmt.Printf("Updated [%s]: %s\n", id, content)
	return nil
}

func cmdRelated(c *internal.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: memo related <id> [limit]")
	}

	id := args[0]
	limit := 5
	if len(args) > 1 {
		if l, err := strconv.Atoi(args[1]); err == nil {
			limit = l
		}
	}

	// Get the embedding for this memory
	embedding, err := c.GetEmbeddingByID(id)
	if err != nil {
		return fmt.Errorf("memory not indexed: %s (run 'memo reindex')", id)
	}

	// Find similar (limit+1 to exclude self)
	results, err := c.Similar(embedding, limit+1, "")
	if err != nil {
		return err
	}

	fmt.Printf("Related to [%s]:\n\n", id)
	for _, r := range results {
		if r.Memory.ID == id {
			continue // skip self
		}
		fmt.Printf("[%s] (%s) (%s) %s\n", r.Memory.ID, r.Score, r.Memory.Type, r.Memory.Content)
	}
	return nil
}

func cmdReindex(c *internal.Client) error {
	fmt.Println("Reindexing all memories...")

	// Delete existing vector set
	c.DeleteVectorSet()

	ids, err := c.GetAllMemoryIDs()
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		fmt.Println("No memories to index.")
		return nil
	}

	count := 0
	for _, id := range ids {
		memo, err := c.Get(id)
		if err != nil {
			continue
		}

		fmt.Printf("  %s: %.50s...\n", id, memo.Content)

		embedding, err := internal.GetDocumentEmbedding(memo.Content)
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}

		if err := c.EmbedMemory(id, embedding); err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		count++
	}

	fmt.Printf("\nIndexed %d memories.\n", count)
	return nil
}

func cmdStats(c *internal.Client) error {
	fmt.Println("Memory Statistics")
	fmt.Println("=================")

	stats, err := c.Stats()
	if err != nil {
		return err
	}

	for _, t := range []string{"fact", "context", "learned", "preference"} {
		fmt.Printf("%-12s %d\n", t+":", stats[t])
	}

	fmt.Println()
	fmt.Printf("Total: %d\n", stats["total"])
	return nil
}

func cmdProjects(c *internal.Client) error {
	projects, err := c.Projects()
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects with memories yet.")
		return nil
	}

	fmt.Println("Projects")
	fmt.Println("========")
	for name, count := range projects {
		fmt.Printf("%-20s %d memories\n", name, count)
	}
	return nil
}

func printHelp() {
	fmt.Println(`memo - Claude's persistent memory system

Commands:
  init                              Initialize the search index
  remember <type> <content> [--tags t1,t2]  Store a memory
  recall <query> [limit]            Search memories (full-text)
  similar <query> [--here] [--limit N]  Semantic search (--here = this project)
  context [limit]                   Show memories for current project
  list [--type TYPE] [--project P] [--here]  List memories with filters
  get <id>                          Get a specific memory
  update <id> <content>             Update a memory's content
  tag <id> <tag>                    Add a tag to a memory
  related <id> [limit]              Find memories similar to one
  forget <id>                       Delete a memory
  reindex                           Generate embeddings for all memories
  stats                             Show memory statistics
  projects                          List all projects with memory counts

Types: fact, context, learned, preference

Examples:
  memo remember fact "User prefers vim keybindings" --tags user,editor
  memo recall "vim"
  memo similar "editor preferences"
  memo list --type preference
  memo get abc123`)
}
