package db

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"helios/internal/config"
)

// ConnectionManager manages a set of named pgxpool connection pools.
type ConnectionManager struct {
	mu    sync.RWMutex
	pools map[string]*pgxpool.Pool
}

// NewConnectionManager creates a ready-to-use ConnectionManager.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		pools: make(map[string]*pgxpool.Pool),
	}
}

// Connect returns an existing pool for cfg.ID or creates a new one.
// Pool creation happens outside the lock to avoid blocking other callers during network I/O.
func (cm *ConnectionManager) Connect(ctx context.Context, cfg config.ConnectionConfig) (*pgxpool.Pool, error) {
	// Fast path: check if pool already exists.
	cm.mu.RLock()
	if p, ok := cm.pools[cfg.ID]; ok {
		cm.mu.RUnlock()
		return p, nil
	}
	cm.mu.RUnlock()

	// Create pool outside the lock to avoid blocking readers during network I/O.
	dsn := buildDSN(cfg)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: connect %q: %w", cfg.Name, err)
	}

	// Verify the connection works immediately.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping %q: %w", cfg.Name, err)
	}

	// Store the pool. If another goroutine raced us, close ours and use theirs.
	cm.mu.Lock()
	if p, ok := cm.pools[cfg.ID]; ok {
		cm.mu.Unlock()
		pool.Close()
		return p, nil
	}
	cm.pools[cfg.ID] = pool
	cm.mu.Unlock()

	return pool, nil
}

// Disconnect closes and removes the pool for the given ID.
// It is a no-op if the ID is not connected.
func (cm *ConnectionManager) Disconnect(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if p, ok := cm.pools[id]; ok {
		p.Close()
		delete(cm.pools, id)
	}
	return nil
}

// Pool returns the pool for the given ID, or nil if not connected.
func (cm *ConnectionManager) Pool(id string) *pgxpool.Pool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.pools[id]
}

// IsConnected reports whether a pool exists for the given ID.
func (cm *ConnectionManager) IsConnected(id string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, ok := cm.pools[id]
	return ok
}

// CloseAll closes every pool and clears the internal map.
func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, p := range cm.pools {
		p.Close()
		delete(cm.pools, id)
	}
}

// buildDSN constructs a postgres:// DSN from a ConnectionConfig.
func buildDSN(cfg config.ConnectionConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.DBName,
	}
	if cfg.SSLMode != "" {
		q := u.Query()
		q.Set("sslmode", cfg.SSLMode)
		u.RawQuery = q.Encode()
	}
	return u.String()
}
