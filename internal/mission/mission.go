// Package mission provides persistent multi-goal mission control for BITTU
// CHAUHAN. It stores active, paused, and completed goals with artifacts and a
// journal so autonomous goal mode can resume real work across sessions.
package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

type JournalEntry struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

type Mission struct {
	ID        int            `json:"id"`
	Goal      string         `json:"goal"`
	Status    Status         `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Artifacts []string       `json:"artifacts,omitempty"`
	Journal   []JournalEntry `json:"journal,omitempty"`
}

type Store struct {
	NextID   int       `json:"next_id"`
	Missions []Mission `json:"missions"`
}

func path(root string) string { return filepath.Join(root, ".zed-missions.json") }

func Load(root string) (*Store, error) {
	buf, err := os.ReadFile(path(root))
	if os.IsNotExist(err) { return &Store{NextID: 1}, nil }
	if err != nil { return nil, err }
	var s Store
	if err := json.Unmarshal(buf, &s); err != nil { return nil, err }
	if s.NextID <= 0 { s.NextID = 1; for _, m := range s.Missions { if m.ID >= s.NextID { s.NextID = m.ID + 1 } } }
	return &s, nil
}

func (s *Store) Save(root string) error {
	if err := os.MkdirAll(root, 0755); err != nil { return err }
	buf, _ := json.MarshalIndent(s, "", "  ")
	tmp := path(root) + ".tmp"
	if err := os.WriteFile(tmp, buf, 0644); err != nil { return err }
	return os.Rename(tmp, path(root))
}

func Start(root, goal string) (*Mission, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" { return nil, fmt.Errorf("goal is required") }
	s, err := Load(root); if err != nil { return nil, err }
	now := time.Now().UTC()
	m := Mission{ID: s.NextID, Goal: goal, Status: StatusActive, CreatedAt: now, UpdatedAt: now, Journal: []JournalEntry{{Time: now, Text: "mission started"}}}
	s.NextID++
	for i := range s.Missions { if s.Missions[i].Status == StatusActive { s.Missions[i].Status = StatusPaused; s.Missions[i].UpdatedAt = now; s.Missions[i].Journal = append(s.Missions[i].Journal, JournalEntry{Time: now, Text: "auto-paused because mission #"+fmt.Sprint(m.ID)+" started"}) } }
	s.Missions = append(s.Missions, m)
	if err := s.Save(root); err != nil { return nil, err }
	return &m, nil
}

func SetStatus(root string, id int, st Status, note string) (*Mission, error) {
	s, err := Load(root); if err != nil { return nil, err }
	if id <= 0 { return nil, fmt.Errorf("mission id is required") }
	now := time.Now().UTC()
	for i := range s.Missions {
		if s.Missions[i].ID == id {
			if st == StatusActive { for j := range s.Missions { if s.Missions[j].Status == StatusActive && s.Missions[j].ID != id { s.Missions[j].Status = StatusPaused; s.Missions[j].UpdatedAt = now } } }
			s.Missions[i].Status = st
			s.Missions[i].UpdatedAt = now
			if note == "" { note = "status changed to " + string(st) }
			s.Missions[i].Journal = append(s.Missions[i].Journal, JournalEntry{Time: now, Text: note})
			if err := s.Save(root); err != nil { return nil, err }
			return &s.Missions[i], nil
		}
	}
	return nil, fmt.Errorf("mission #%d not found", id)
}

func AddJournal(root string, id int, text string) (*Mission, error) {
	s, err := Load(root); if err != nil { return nil, err }
	now := time.Now().UTC()
	for i := range s.Missions { if s.Missions[i].ID == id { s.Missions[i].Journal = append(s.Missions[i].Journal, JournalEntry{Time: now, Text: text}); s.Missions[i].UpdatedAt = now; if err := s.Save(root); err != nil { return nil, err }; return &s.Missions[i], nil } }
	return nil, fmt.Errorf("mission #%d not found", id)
}

func AddArtifact(root string, id int, artifact string) (*Mission, error) {
	s, err := Load(root); if err != nil { return nil, err }
	now := time.Now().UTC()
	for i := range s.Missions { if s.Missions[i].ID == id { s.Missions[i].Artifacts = append(s.Missions[i].Artifacts, artifact); s.Missions[i].Journal = append(s.Missions[i].Journal, JournalEntry{Time: now, Text: "artifact added: "+artifact}); s.Missions[i].UpdatedAt = now; if err := s.Save(root); err != nil { return nil, err }; return &s.Missions[i], nil } }
	return nil, fmt.Errorf("mission #%d not found", id)
}

func List(root string) (*Store, error) { s, err := Load(root); if err != nil { return nil, err }; sort.Slice(s.Missions, func(i,j int) bool { return s.Missions[i].ID < s.Missions[j].ID }); return s, nil }

func Render(s *Store) string {
	var b strings.Builder
	b.WriteString("🚀 Mission Control\n")
	if len(s.Missions) == 0 { b.WriteString("  No missions yet. Start with /mission start <goal>\n"); return b.String() }
	for _, m := range s.Missions {
		fmt.Fprintf(&b, "  #%d [%s] %s\n", m.ID, strings.ToUpper(string(m.Status)), m.Goal)
		fmt.Fprintf(&b, "      updated: %s artifacts:%d journal:%d\n", m.UpdatedAt.Format(time.RFC3339), len(m.Artifacts), len(m.Journal))
		if len(m.Artifacts) > 0 { fmt.Fprintf(&b, "      latest artifact: %s\n", m.Artifacts[len(m.Artifacts)-1]) }
	}
	return b.String()
}
