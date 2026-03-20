package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// ConnectionConfig holds the parameters needed to connect to a PostgreSQL database.
type ConnectionConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// AppConfig is the top-level configuration containing saved connections.
type AppConfig struct {
	Connections []ConnectionConfig `json:"connections"`
}

// DefaultPath returns the default config file path (~/.config/helios/connections.json).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: user config dir: %w", err)
	}
	return filepath.Join(dir, "helios", "connections.json"), nil
}

// Load reads and unmarshals a JSON config file at path.
// If the file does not exist, the config is left empty (no error).
func (c *AppConfig) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("config: read file: %w", err)
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("config: unmarshal: %w", err)
	}
	return nil
}

// Save marshals the config to JSON and writes it to path.
// Parent directories are created if needed. The file is written with 0600 permissions.
func (c *AppConfig) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("config: create dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("config: write file: %w", err)
	}
	return nil
}

// Add appends a connection to the config. If cfg.ID is empty, a new UUID is generated.
func (c *AppConfig) Add(cfg ConnectionConfig) {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	c.Connections = append(c.Connections, cfg)
}

// Update replaces an existing connection by ID. Returns an error if the ID is not found.
func (c *AppConfig) Update(cfg ConnectionConfig) error {
	for i, conn := range c.Connections {
		if conn.ID == cfg.ID {
			c.Connections[i] = cfg
			return nil
		}
	}
	return fmt.Errorf("config: connection %q not found", cfg.ID)
}

// Remove deletes a connection by ID. Returns an error if the ID is not found.
func (c *AppConfig) Remove(id string) error {
	for i, conn := range c.Connections {
		if conn.ID == id {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("config: connection %q not found", id)
}

// FindByID looks up a connection by ID. Returns an error if the ID is not found.
func (c *AppConfig) FindByID(id string) (*ConnectionConfig, error) {
	for i, conn := range c.Connections {
		if conn.ID == id {
			return &c.Connections[i], nil
		}
	}
	return nil, fmt.Errorf("config: connection %q not found", id)
}
