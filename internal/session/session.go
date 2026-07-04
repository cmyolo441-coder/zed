// Package session persists ZED conversations to disk so users can resume work
// across restarts. Sessions are stored as JSON files under the config dir, one
// file per session, with metadata for listing and quick resume.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cmyolo441-coder/zed/internal/llm"
)

// Session is a saved conversation with metadata.
type Session struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Provider  string        `json:"provider"`
	Model     string        `json:"model"`
	WorkDir   string        `json:"work_dir"`
	Created   time.Time     `json:"created"`
	Updated   time.Time     `json:"updated"`
	Messages  []llm.Message `json:"messages"`
	Tokens    int           `json:"tokens"`
}

// Meta is a lightweight summary used for listing sessions without loading all
// message content.
type Meta struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Model    string    `json:"model"`
	WorkDir  string    `json:"work_dir"`
	Updated  time.Time `json:"updated"`
	Messages int       `json:"messages"`
}

// Store manages session files on disk.
type Store struct {
	dir string
}

// NewStore creates a session store rooted at the given directory (created if
// missing). Pass an empty dir to use the default location.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		dir = DefaultDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// DefaultDir returns the standard session storage directory.
func DefaultDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "zed", "sessions")
}

// NewSession creates a fresh, empty session with a generated ID.
func NewSession(provider, model, workDir string) *Session {
	now := time.Now()
	return &Session{
		ID:       generateID(now),
		Title:    "Untitled session",
		Provider: provider,
		Model:    model,
		WorkDir:  workDir,
		Created:  now,
		Updated:  now,
	}
}

// Save writes the session to disk atomically (write temp then rename).
func (s *Store) Save(sess *Session) error {
	sess.Updated = time.Now()
	if sess.Title == "Untitled session" {
		sess.Title = deriveTitle(sess.Messages)
	}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	path := s.path(sess.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a session by ID.
func (s *Store) Load(id string) (*Session, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// Delete removes a session file.
func (s *Store) Delete(id string) error {
	return os.Remove(s.path(id))
}

// List returns metadata for all saved sessions, newest first.
func (s *Store) List() ([]Meta, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var metas []Meta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var sess Session
		if json.Unmarshal(data, &sess) != nil {
			continue
		}
		metas = append(metas, Meta{
			ID:       sess.ID,
			Title:    sess.Title,
			Model:    sess.Model,
			WorkDir:  sess.WorkDir,
			Updated:  sess.Updated,
			Messages: len(sess.Messages),
		})
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Updated.After(metas[j].Updated)
	})
	return metas, nil
}

// Latest returns the most recently updated session, or nil if none exist.
func (s *Store) Latest() (*Session, error) {
	metas, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, nil
	}
	return s.Load(metas[0].ID)
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func generateID(t time.Time) string {
	return t.Format("20060102-150405")
}

// deriveTitle builds a short title from the first user message.
func deriveTitle(msgs []llm.Message) string {
	for _, m := range msgs {
		if m.Role == llm.RoleUser && strings.TrimSpace(m.Content) != "" {
			title := strings.TrimSpace(m.Content)
			title = strings.ReplaceAll(title, "\n", " ")
			if len(title) > 60 {
				title = title[:60] + "…"
			}
			return title
		}
	}
	return "Untitled session"
}
