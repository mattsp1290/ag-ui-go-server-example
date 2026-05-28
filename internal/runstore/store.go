// Package runstore is an in-memory store of paused agent runs, keyed by
// threadID+runID, so an interrupted run can be resumed by a later request.
//
// This is deliberately process-local and non-durable — it exists to demonstrate
// the AG-UI interrupt -> resume cycle for the local proof, not for production.
package runstore

import (
	"sync"

	"github.com/cloudwego/eino/schema"
)

// Saved is the state captured when a run pauses on a tool-approval interrupt.
type Saved struct {
	// Messages is the full eino conversation up to and including the assistant
	// message that proposed the pending tool calls. The assistant message is kept
	// whole (including Extra, which carries reasoning items the codex model needs
	// threaded across turns).
	Messages []*schema.Message
	// Pending are the tool calls awaiting human approval.
	Pending []schema.ToolCall
	// State is the agent state snapshot at pause time.
	State map[string]any
}

// Store is a concurrency-safe map of paused runs.
type Store struct {
	mu sync.Mutex
	m  map[string]*Saved
}

// New creates an empty Store.
func New() *Store {
	return &Store{m: make(map[string]*Saved)}
}

// Key builds the lookup key for a run.
func Key(threadID, runID string) string {
	return threadID + "|" + runID
}

// Save records a paused run.
func (s *Store) Save(key string, saved *Saved) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = saved
}

// Load returns a paused run and whether it was present.
func (s *Store) Load(key string) (*Saved, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	saved, ok := s.m[key]
	return saved, ok
}

// Delete removes a paused run (called once it has been resumed).
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}
