package config

import (
	"os"
	"path/filepath"
	"testing"
)

func sampleConnection(id, name string) ConnectionConfig {
	return ConnectionConfig{
		ID:       id,
		Name:     name,
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		DBName:   "testdb",
		SSLMode:  "disable",
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // returns file path
		wantConns int
		wantErr   bool
	}{
		{
			name: "valid JSON",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.json")
				data := `{"connections":[{"id":"1","name":"dev","host":"localhost","port":5432,"user":"pg","password":"pw","dbname":"db","sslmode":"disable"}]}`
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantConns: 1,
		},
		{
			name: "missing file returns empty config",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nonexistent.json")
			},
			wantConns: 0,
		},
		{
			name: "malformed JSON errors",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "config.json")
				if err := os.WriteFile(path, []byte("{bad json"), 0600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			var cfg AppConfig
			err := cfg.Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Connections) != tt.wantConns {
				t.Fatalf("got %d connections, want %d", len(cfg.Connections), tt.wantConns)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	original := &AppConfig{
		Connections: []ConnectionConfig{
			sampleConnection("id-1", "dev"),
			sampleConnection("id-2", "staging"),
		},
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("permissions = %o, want 0600", perm)
	}

	// Round-trip.
	var loaded AppConfig
	if err := loaded.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Connections) != 2 {
		t.Fatalf("got %d connections, want 2", len(loaded.Connections))
	}
	if loaded.Connections[0].Name != "dev" {
		t.Fatalf("first connection name = %q, want %q", loaded.Connections[0].Name, "dev")
	}
	if loaded.Connections[1].Name != "staging" {
		t.Fatalf("second connection name = %q, want %q", loaded.Connections[1].Name, "staging")
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name   string
		input  ConnectionConfig
		wantID bool // true if we expect a generated UUID
	}{
		{
			name:   "generates UUID when ID is empty",
			input:  ConnectionConfig{Name: "dev", Host: "localhost"},
			wantID: true,
		},
		{
			name:   "keeps provided ID",
			input:  ConnectionConfig{ID: "custom-id", Name: "staging"},
			wantID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg AppConfig
			cfg.Add(tt.input)

			if len(cfg.Connections) != 1 {
				t.Fatalf("got %d connections, want 1", len(cfg.Connections))
			}
			added := cfg.Connections[0]
			if tt.wantID {
				if added.ID == "" {
					t.Fatal("expected generated UUID, got empty")
				}
			} else {
				if added.ID != "custom-id" {
					t.Fatalf("ID = %q, want %q", added.ID, "custom-id")
				}
			}
			if added.Name != tt.input.Name {
				t.Fatalf("Name = %q, want %q", added.Name, tt.input.Name)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "existing ID", id: "id-1", wantErr: false},
		{name: "unknown ID", id: "no-such-id", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AppConfig{
				Connections: []ConnectionConfig{sampleConnection("id-1", "dev")},
			}
			updated := sampleConnection(tt.id, "updated")
			err := cfg.Update(updated)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Connections[0].Name != "updated" {
				t.Fatalf("Name = %q, want %q", cfg.Connections[0].Name, "updated")
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		wantLen int
	}{
		{name: "existing ID", id: "id-1", wantErr: false, wantLen: 1},
		{name: "unknown ID", id: "no-such-id", wantErr: true, wantLen: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AppConfig{
				Connections: []ConnectionConfig{
					sampleConnection("id-1", "dev"),
					sampleConnection("id-2", "staging"),
				},
			}
			err := cfg.Remove(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Connections) != tt.wantLen {
				t.Fatalf("got %d connections, want %d", len(cfg.Connections), tt.wantLen)
			}
		})
	}
}

func TestFindByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "existing ID", id: "id-1", wantErr: false},
		{name: "unknown ID", id: "no-such-id", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AppConfig{
				Connections: []ConnectionConfig{
					sampleConnection("id-1", "dev"),
					sampleConnection("id-2", "staging"),
				},
			}
			conn, err := cfg.FindByID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if conn != nil {
					t.Fatal("expected nil connection on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if conn.ID != tt.id {
				t.Fatalf("ID = %q, want %q", conn.ID, tt.id)
			}
		})
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(p) != "connections.json" {
		t.Fatalf("filename = %q, want connections.json", filepath.Base(p))
	}
}
