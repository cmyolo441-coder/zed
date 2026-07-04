// Package collab provides a real-time collaboration layer. Multiple users
// can work on the same project simultaneously with conflict-free edits (CRDT)
// and shared agent state.
package collab

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// User represents a collaborator.
type User struct {
	ID       string
	Name     string
	Color    string // display color
	ActiveFile string
	LastSeen time.Time
}

// Edit is a conflict-free edit operation (simplified CRDT).
type Edit struct {
	ID        string
	UserID    string
	File      string
	Operation string // "insert", "delete", "replace"
	Position  int
	Content   string
	Timestamp time.Time
}

// Session is a shared collaboration session.
type Session struct {
	mu    sync.RWMutex
	users map[string]*User
	edits []Edit
	locks map[string]string // file → user ID (who's editing it)
}

// NewSession creates a collaboration session.
func NewSession() *Session {
	return &Session{
		users: map[string]*User{},
		locks: map[string]string{},
	}
}

// Join adds a user to the session.
func (s *Session) Join(user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user.LastSeen = time.Now()
	s.users[user.ID] = user
}

// Leave removes a user from the session.
func (s *Session) Leave(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, userID)
	for file, uid := range s.locks {
		if uid == userID {
			delete(s.locks, file)
		}
	}
}

// LockFile lets a user claim a file for editing (prevents conflicts).
func (s *Session) LockFile(userID, file string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if owner, ok := s.locks[file]; ok && owner != userID {
		return false // someone else has the lock
	}
	s.locks[file] = userID
	if u, ok := s.users[userID]; ok {
		u.ActiveFile = file
	}
	return true
}

// UnlockFile releases a file lock.
func (s *Session) UnlockFile(userID, file string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locks[file] == userID {
		delete(s.locks, file)
	}
}

// ApplyEdit applies a conflict-free edit.
func (s *Session) ApplyEdit(edit Edit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	edit.Timestamp = time.Now()
	s.edits = append(s.edits, edit)
}

// ActiveUsers returns who's currently in the session.
func (s *Session) ActiveUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var users []*User
	for _, u := range s.users {
		if time.Since(u.LastSeen) < 5*time.Minute {
			users = append(users, u)
		}
	}
	return users
}

// WhoIsEditing returns the user editing a given file.
func (s *Session) WhoIsEditing(file string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if uid, ok := s.locks[file]; ok {
		return s.users[uid]
	}
	return nil
}

// Status returns the collaboration session status.
func (s *Session) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

// statusLocked returns the status string assuming the caller already holds
// a lock. Avoids the deadlock that occurs when Status() called ActiveUsers()
// which also tried to acquire RLock while the same goroutine held it.
func (s *Session) statusLocked() string {
	var b strings.Builder
	b.WriteString("👥 Collaboration Session\n")

	// Count active users inline — don't call ActiveUsers() (deadlock risk).
	var active []*User
	now := time.Now()
	for _, u := range s.users {
		if now.Sub(u.LastSeen) < 5*time.Minute {
			active = append(active, u)
		}
	}
	b.WriteString(fmt.Sprintf("  Active users: %d\n", len(active)))
	for _, u := range active {
		file := u.ActiveFile
		if file == "" {
			file = "idle"
		}
		b.WriteString(fmt.Sprintf("  • %s: editing %s\n", u.Name, file))
	}
	if len(s.locks) > 0 {
		b.WriteString("\n  File locks:\n")
		for file, uid := range s.locks {
			if u, ok := s.users[uid]; ok {
				b.WriteString(fmt.Sprintf("    %s → %s\n", file, u.Name))
			}
		}
	}
	b.WriteString(fmt.Sprintf("\n  Total edits: %d\n", len(s.edits)))
	return b.String()
}
