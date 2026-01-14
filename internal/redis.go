package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	IndexName = "memo_idx"
	VectorSet = "memovecs"
)

var ctx = context.Background()

// Memory represents a stored memory
type Memory struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	Created     string   `json:"created"`
	Accessed    string   `json:"accessed"`
	AccessCount int      `json:"access_count"`
}

// Client wraps Redis connection
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis client
func NewClient() *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	return &Client{rdb: rdb}
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// GenID generates a short unique ID
func GenID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Now returns current ISO timestamp
func Now() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// Init creates the search index
func (c *Client) Init() error {
	// Drop existing index (keep documents)
	c.rdb.Do(ctx, "FT.DROPINDEX", IndexName).Err()

	// Create the search index
	_, err := c.rdb.Do(ctx, "FT.CREATE", IndexName,
		"ON", "JSON",
		"PREFIX", "1", "memo:",
		"SCHEMA",
		"$.content", "AS", "content", "TEXT",
		"$.type", "AS", "type", "TAG",
		"$.tags[*]", "AS", "tags", "TAG",
	).Result()

	return err
}

// Remember stores a new memory
func (c *Client) Remember(memType, content string, tags []string, project string) (*Memory, error) {
	id := GenID()
	ts := Now()

	// Always include project tag
	allTags := append([]string{"project:" + project}, tags...)

	memo := Memory{
		ID:          id,
		Type:        memType,
		Content:     content,
		Tags:        allTags,
		Created:     ts,
		Accessed:    ts,
		AccessCount: 0,
	}

	jsonData, err := json.Marshal(memo)
	if err != nil {
		return nil, err
	}

	_, err = c.rdb.Do(ctx, "JSON.SET", "memo:"+id, "$", string(jsonData)).Result()
	if err != nil {
		return nil, err
	}

	return &memo, nil
}

// EmbedMemory adds a memory's embedding to the vector set
func (c *Client) EmbedMemory(id string, embedding []float64) error {
	args := []interface{}{"VADD", VectorSet, "VALUES", len(embedding)}
	for _, v := range embedding {
		args = append(args, v)
	}
	args = append(args, id)

	_, err := c.rdb.Do(ctx, args...).Result()
	return err
}

// Recall searches memories using full-text search
func (c *Client) Recall(query string, limit int) ([]Memory, error) {
	result, err := c.rdb.Do(ctx, "FT.SEARCH", IndexName, query,
		"LIMIT", "0", fmt.Sprint(limit),
		"RETURN", "1", "$",
	).Result()
	if err != nil {
		return nil, err
	}

	return parseSearchResults(result)
}

// List returns memories with optional filters
func (c *Client) List(typeFilter, tagFilter string, limit int) ([]Memory, error) {
	query := "*"
	if typeFilter != "" && tagFilter != "" {
		query = fmt.Sprintf("@type:{%s} @tags:{%s}", typeFilter, tagFilter)
	} else if typeFilter != "" {
		query = fmt.Sprintf("@type:{%s}", typeFilter)
	} else if tagFilter != "" {
		query = fmt.Sprintf("@tags:{%s}", tagFilter)
	}

	result, err := c.rdb.Do(ctx, "FT.SEARCH", IndexName, query,
		"LIMIT", "0", fmt.Sprint(limit),
		"RETURN", "1", "$",
	).Result()
	if err != nil {
		return nil, err
	}

	return parseSearchResults(result)
}

// Context returns memories for the current project
func (c *Client) Context(project string, limit int) ([]Memory, error) {
	// Use wildcard search and filter client-side (colon escaping is problematic)
	result, err := c.rdb.Do(ctx, "FT.SEARCH", IndexName, "@tags:{project*}",
		"LIMIT", "0", "100",
		"RETURN", "1", "$",
	).Result()
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	memos, err := parseSearchResults(result)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Filter by project
	projectTag := "project:" + project
	var filtered []Memory
	for _, m := range memos {
		for _, tag := range m.Tags {
			if tag == projectTag {
				filtered = append(filtered, m)
				break
			}
		}
		if len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

// Get retrieves a specific memory and updates access stats
func (c *Client) Get(id string) (*Memory, error) {
	result, err := c.rdb.Do(ctx, "JSON.GET", "memo:"+id).Result()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	var memo Memory
	if err := json.Unmarshal([]byte(result.(string)), &memo); err != nil {
		return nil, err
	}

	// Update access stats
	c.rdb.Do(ctx, "JSON.NUMINCRBY", "memo:"+id, "$.access_count", 1)
	c.rdb.Do(ctx, "JSON.SET", "memo:"+id, "$.accessed", fmt.Sprintf("\"%s\"", Now()))

	return &memo, nil
}

// AddTag adds a tag to an existing memory
func (c *Client) AddTag(id, tag string) error {
	// Get current memory
	memo, err := c.getMemoryRaw(id)
	if err != nil {
		return err
	}

	// Check if tag already exists
	for _, t := range memo.Tags {
		if t == tag {
			return fmt.Errorf("tag already exists: %s", tag)
		}
	}

	// Append new tag
	memo.Tags = append(memo.Tags, tag)
	tagsJSON, _ := json.Marshal(memo.Tags)

	_, err = c.rdb.Do(ctx, "JSON.SET", "memo:"+id, "$.tags", string(tagsJSON)).Result()
	return err
}

// Update modifies a memory's content and re-embeds it
func (c *Client) Update(id, content string) error {
	// Check memory exists
	_, err := c.Get(id)
	if err != nil {
		return err
	}

	// Update content
	_, err = c.rdb.Do(ctx, "JSON.SET", "memo:"+id, "$.content", fmt.Sprintf("\"%s\"", content)).Result()
	return err
}

// GetEmbedding returns the embedding for a memory ID from the vector set
func (c *Client) GetEmbeddingByID(id string) ([]float64, error) {
	result, err := c.rdb.Do(ctx, "VEMB", VectorSet, id).Result()
	if err != nil {
		return nil, err
	}

	arr, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected VEMB result type: %T", result)
	}

	embedding := make([]float64, len(arr))
	for i, v := range arr {
		switch val := v.(type) {
		case float64:
			embedding[i] = val
		case string:
			fmt.Sscanf(val, "%f", &embedding[i])
		}
	}
	return embedding, nil
}

// Forget deletes a memory
func (c *Client) Forget(id string) error {
	result, err := c.rdb.Do(ctx, "JSON.DEL", "memo:"+id).Result()
	if err != nil {
		return err
	}
	if result.(int64) == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}
	return nil
}

// Projects returns all projects with their memory counts
func (c *Client) Projects() (map[string]int, error) {
	// Get all memories with project tags
	result, err := c.rdb.Do(ctx, "FT.SEARCH", IndexName, "@tags:{project*}",
		"LIMIT", "0", "1000",
		"RETURN", "1", "$.tags",
	).Result()
	if err != nil {
		return nil, err
	}

	projects := make(map[string]int)

	// Parse results to extract project tags
	switch res := result.(type) {
	case map[interface{}]interface{}:
		results, ok := res["results"]
		if !ok {
			return projects, nil
		}
		resultsArr, ok := results.([]interface{})
		if !ok {
			return projects, nil
		}
		for _, item := range resultsArr {
			itemMap, ok := item.(map[interface{}]interface{})
			if !ok {
				continue
			}
			extraAttrs, ok := itemMap["extra_attributes"]
			if !ok {
				continue
			}
			attrsMap, ok := extraAttrs.(map[interface{}]interface{})
			if !ok {
				continue
			}
			tagsStr, ok := attrsMap["$.tags"].(string)
			if !ok {
				continue
			}
			var tags []string
			json.Unmarshal([]byte(tagsStr), &tags)
			for _, tag := range tags {
				if len(tag) > 8 && tag[:8] == "project:" {
					projects[tag[8:]]++
				}
			}
		}
	}

	return projects, nil
}

// Stats returns memory statistics
func (c *Client) Stats() (map[string]int, error) {
	stats := make(map[string]int)
	types := []string{"fact", "context", "learned", "preference"}

	for _, t := range types {
		result, err := c.rdb.Do(ctx, "FT.SEARCH", IndexName,
			fmt.Sprintf("@type:{%s}", t),
			"LIMIT", "0", "0",
		).Result()
		if err != nil {
			continue
		}
		stats[t] = parseSearchCount(result)
	}

	// Total
	result, err := c.rdb.Do(ctx, "FT.SEARCH", IndexName, "*", "LIMIT", "0", "0").Result()
	if err == nil {
		stats["total"] = parseSearchCount(result)
	}

	return stats, nil
}

// parseSearchCount extracts count from FT.SEARCH result
func parseSearchCount(result interface{}) int {
	switch res := result.(type) {
	case map[interface{}]interface{}:
		// RESP3 format
		if total, ok := res["total_results"]; ok {
			if count, ok := total.(int64); ok {
				return int(count)
			}
		}
	case []interface{}:
		// RESP2 format
		if len(res) > 0 {
			if count, ok := res[0].(int64); ok {
				return int(count)
			}
		}
	case int64:
		return int(res)
	}
	return 0
}

// Similar finds semantically similar memories
func (c *Client) Similar(embedding []float64, limit int, project string) ([]SimilarResult, error) {
	// Check if vector set exists
	_, err := c.rdb.Do(ctx, "VCARD", VectorSet).Result()
	if err != nil {
		return nil, fmt.Errorf("no embeddings found - run 'memo reindex' first")
	}

	// Build VSIM command
	fetchLimit := limit
	if project != "" {
		fetchLimit = limit * 3 // Fetch more to filter
	}

	args := []interface{}{"VSIM", VectorSet, "VALUES", len(embedding)}
	for _, v := range embedding {
		args = append(args, v)
	}
	args = append(args, "COUNT", fetchLimit, "WITHSCORES")

	result, err := c.rdb.Do(ctx, args...).Result()
	if err != nil {
		return nil, err
	}

	// Parse VSIM results
	type vsimItem struct {
		id    string
		score string
	}
	var items []vsimItem

	switch res := result.(type) {
	case []interface{}:
		// Array format: [id1, score1, id2, score2, ...]
		for i := 0; i < len(res)-1; i += 2 {
			var id, score string
			switch v := res[i].(type) {
			case string:
				id = v
			case []byte:
				id = string(v)
			}
			switch v := res[i+1].(type) {
			case string:
				score = v
			case float64:
				score = fmt.Sprintf("%.2f", v)
			}
			if id != "" {
				items = append(items, vsimItem{id, score})
			}
		}
	case map[interface{}]interface{}:
		// Map format: {id: score, id: score, ...}
		for k, v := range res {
			var id, score string
			switch key := k.(type) {
			case string:
				id = key
			}
			switch val := v.(type) {
			case float64:
				score = fmt.Sprintf("%.2f", val)
			case string:
				score = val
			}
			if id != "" {
				items = append(items, vsimItem{id, score})
			}
		}
	default:
		return nil, fmt.Errorf("unexpected VSIM result type: %T", result)
	}

	projectTag := "project:" + project
	var results []SimilarResult
	for _, item := range items {
		if len(results) >= limit {
			break
		}

		// Get memory details
		memo, err := c.getMemoryRaw(item.id)
		if err != nil {
			continue
		}

		// Filter by project if specified
		if project != "" {
			found := false
			for _, tag := range memo.Tags {
				if tag == projectTag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		results = append(results, SimilarResult{
			Memory: *memo,
			Score:  item.score,
		})
	}

	return results, nil
}

// SimilarResult holds a memory with its similarity score
type SimilarResult struct {
	Memory Memory
	Score  string
}

// getMemoryRaw retrieves a memory without updating access stats
func (c *Client) getMemoryRaw(id string) (*Memory, error) {
	result, err := c.rdb.Do(ctx, "JSON.GET", "memo:"+id).Result()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	var memo Memory
	if err := json.Unmarshal([]byte(result.(string)), &memo); err != nil {
		return nil, err
	}
	return &memo, nil
}

// GetAllMemoryIDs returns all memory IDs for reindexing
func (c *Client) GetAllMemoryIDs() ([]string, error) {
	var ids []string
	iter := c.rdb.Scan(ctx, 0, "memo:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		ids = append(ids, key[5:]) // Remove "memo:" prefix
	}
	return ids, iter.Err()
}

// DeleteVectorSet removes all vectors for reindexing
func (c *Client) DeleteVectorSet() error {
	return c.rdb.Del(ctx, VectorSet).Err()
}

// parseSearchResults parses FT.SEARCH results into Memory structs
// Handles both RESP2 (array) and RESP3 (map) formats
func parseSearchResults(result interface{}) ([]Memory, error) {
	var memos []Memory

	switch res := result.(type) {
	case map[interface{}]interface{}:
		// RESP3 format: map with "results" key
		results, ok := res["results"]
		if !ok {
			return nil, nil
		}
		resultsArr, ok := results.([]interface{})
		if !ok {
			return nil, nil
		}
		for _, item := range resultsArr {
			itemMap, ok := item.(map[interface{}]interface{})
			if !ok {
				continue
			}
			extraAttrs, ok := itemMap["extra_attributes"]
			if !ok {
				continue
			}
			attrsMap, ok := extraAttrs.(map[interface{}]interface{})
			if !ok {
				continue
			}
			jsonStr, ok := attrsMap["$"].(string)
			if !ok {
				continue
			}
			var memo Memory
			if err := json.Unmarshal([]byte(jsonStr), &memo); err != nil {
				continue
			}
			memos = append(memos, memo)
		}

	case []interface{}:
		// RESP2 format: [count, key1, fields1, key2, fields2, ...]
		for i := 1; i < len(res); i += 2 {
			if i+1 >= len(res) {
				break
			}
			fields, ok := res[i+1].([]interface{})
			if !ok || len(fields) < 2 {
				continue
			}
			jsonStr, ok := fields[1].(string)
			if !ok {
				continue
			}
			var memo Memory
			if err := json.Unmarshal([]byte(jsonStr), &memo); err != nil {
				continue
			}
			memos = append(memos, memo)
		}
	}

	return memos, nil
}
