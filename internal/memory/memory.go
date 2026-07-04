// Package memory provides persistent storage for the agent's long-term
// knowledge. It remembers facts about the codebase, user preferences,
// and patterns discovered across sessions — so the agent doesn't have to
// relearn everything from scratch each time.
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry is a single memory record.
type Entry struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`  // "fact", "preference", "pattern", "decision"
	Key       string    `json:"key"`       // short identifier
	Value     string    `json:"value"`     // the memory content
	Tags      []string  `json:"tags"`      // searchable tags
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Hits      int       `json:"hits"` // how many times recalled
}

// Store is the persistent memory store. It saves to a JSON file in the
// user's config directory and caches entries in memory for fast access.
type Store struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	path    string
}

// New creates or opens a memory store at the default location.
func New() *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	path := filepath.Join(dir, "zed", "memory.json")
	s := &Store{
		entries: map[string]*Entry{},
		path:    path,
	}
	s.load()
	return s
}

// load reads the memory file if it exists.
func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return // no file yet — fresh memory
	}
	var entries []*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range entries {
		s.entries[e.ID] = e
	}
}

// save persists all entries to disk.
func (s *Store) save() error {
	s.mu.RLock()
	entries := make([]*Entry, 0, len(s.entries))
	for _, e := range s.entries {
		entries = append(entries, e)
	}
	s.mu.RUnlock()

	// Sort by created time for stable output.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Remember stores a new memory or updates an existing one by key.
func (s *Store) Remember(category, key, value string, tags ...string) error {
	s.mu.Lock()
	id := category + ":" + key
	now := time.Now()
	if e, ok := s.entries[id]; ok {
		e.Value = value
		e.UpdatedAt = now
		e.Tags = tags
		s.mu.Unlock()
		return s.save()
	}
	s.entries[id] = &Entry{
		ID:        id,
		Category:  category,
		Key:       key,
		Value:     value,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Unlock()
	return s.save()
}

// Recall retrieves a specific memory by category and key.
func (s *Store) Recall(category, key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := category + ":" + key
	e, ok := s.entries[id]
	if !ok {
		return "", false
	}
	e.Hits++
	return e.Value, true
}

// Search returns memories matching the given text in key, value, or tags.
func (s *Store) Search(query string, limit int) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var results []*Entry
	for _, e := range s.entries {
		if strings.Contains(strings.ToLower(e.Key), q) ||
			strings.Contains(strings.ToLower(e.Value), q) ||
			strings.Contains(strings.ToLower(e.Category), q) {
			results = append(results, e)
			continue
		}
		for _, t := range e.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				results = append(results, e)
				break
			}
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Hits > results[j].Hits })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// RecallCategory returns all memories in a category.
func (s *Store) RecallCategory(category string) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*Entry
	for _, e := range s.entries {
		if e.Category == category {
			results = append(results, e)
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].CreatedAt.Before(results[j].CreatedAt) })
	return results
}

// Forget removes a memory by category and key.
func (s *Store) Forget(category, key string) error {
	s.mu.Lock()
	id := category + ":" + key
	delete(s.entries, id)
	s.mu.Unlock()
	return s.save()
}

// All returns all memories sorted by creation time.
func (s *Store) All() []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]*Entry, 0, len(s.entries))
	for _, e := range s.entries {
		results = append(results, e)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].CreatedAt.Before(results[j].CreatedAt) })
	return results
}

// ContextForPrompt returns a formatted summary of relevant memories to inject
// into the system prompt so the agent has context from past sessions.
func (s *Store) ContextForPrompt(maxEntries int) string {
	entries := s.All()
	if len(entries) == 0 {
		return ""
	}
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	var b strings.Builder
	b.WriteString("\n\n## Persistent Memory (from past sessions)\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", e.Category, e.Key, e.Value))
	}
	return b.String()
}
