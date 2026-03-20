package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryEntry is a single recorded query.
type HistoryEntry struct {
	SQL       string    `json:"sql"`
	Timestamp time.Time `json:"timestamp"`
	ConnName  string    `json:"connection"`
}

// QueryHistory stores executed queries with no limit.
type QueryHistory struct {
	mu      sync.Mutex
	entries []HistoryEntry
	path    string
}

// NewQueryHistory creates a history backed by a JSON file next to the config.
func NewQueryHistory(configPath string) *QueryHistory {
	dir := filepath.Dir(configPath)
	h := &QueryHistory{
		path: filepath.Join(dir, "query_history.json"),
	}
	h.load()
	return h
}

// Add records a query execution. Save is async to avoid blocking.
func (h *QueryHistory) Add(sql, connName string) {
	h.mu.Lock()
	h.entries = append(h.entries, HistoryEntry{
		SQL:       sql,
		Timestamp: time.Now(),
		ConnName:  connName,
	})
	h.mu.Unlock()
	go h.save()
}

// Entries returns a copy of all entries, newest first.
func (h *QueryHistory) Entries() []HistoryEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]HistoryEntry, len(h.entries))
	for i, e := range h.entries {
		result[len(h.entries)-1-i] = e
	}
	return result
}

// Clear removes all history entries.
func (h *QueryHistory) Clear() {
	h.mu.Lock()
	h.entries = nil
	h.mu.Unlock()
	h.save()
}

func (h *QueryHistory) load() {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	h.mu.Lock()
	_ = json.Unmarshal(data, &h.entries)
	h.mu.Unlock()
}

func (h *QueryHistory) save() {
	h.mu.Lock()
	// Copy under lock, marshal outside lock.
	cp := make([]HistoryEntry, len(h.entries))
	copy(cp, h.entries)
	h.mu.Unlock()

	data, err := json.Marshal(cp)
	if err != nil {
		return
	}
	dir := filepath.Dir(h.path)
	_ = os.MkdirAll(dir, 0700)
	_ = os.WriteFile(h.path, data, 0600)
}
