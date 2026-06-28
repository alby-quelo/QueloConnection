package registry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nossh/nossh/internal/protocol"
	_ "modernc.org/sqlite"
)

type AgentRecord struct {
	UUID      string               `json:"uuid"`
	Code      string               `json:"code"`
	Hostname  string               `json:"hostname"`
	Name      string               `json:"name"`
	Status    protocol.AgentStatus `json:"status"`
	CreatedAt time.Time            `json:"created_at"`
	LastSeen  time.Time            `json:"last_seen"`
	Online    bool                 `json:"online"`
}

type Registry struct {
	db *sql.DB
}

func Open(dataDir string) (*Registry, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dataDir, "registry.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open registry: %w", err)
	}
	r := &Registry{db: db}
	if err := r.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return r, nil
}

func (r *Registry) Close() error {
	return r.db.Close()
}

func (r *Registry) migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			uuid TEXT PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			hostname TEXT NOT NULL DEFAULT '',
			name TEXT UNIQUE,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TEXT NOT NULL,
			last_seen TEXT NOT NULL
		);
	`)
	return err
}

func (r *Registry) UpsertAgent(uuid, code, hostname string) (*AgentRecord, error) {
	now := time.Now().UTC()
	_, err := r.db.Exec(`
		INSERT INTO agents (uuid, code, hostname, status, created_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			hostname = excluded.hostname,
			last_seen = excluded.last_seen
	`, uuid, code, hostname, protocol.StatusPending, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("upsert agent: %w", err)
	}
	return r.GetByUUID(uuid)
}

func (r *Registry) Touch(uuid string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`UPDATE agents SET last_seen = ? WHERE uuid = ?`, now, uuid)
	return err
}

func (r *Registry) GetByUUID(uuid string) (*AgentRecord, error) {
	row := r.db.QueryRow(`
		SELECT uuid, code, hostname, COALESCE(name, ''), status, created_at, last_seen
		FROM agents WHERE uuid = ?
	`, uuid)
	return scanAgent(row, false)
}

func (r *Registry) GetByCode(code string) (*AgentRecord, error) {
	row := r.db.QueryRow(`
		SELECT uuid, code, hostname, COALESCE(name, ''), status, created_at, last_seen
		FROM agents WHERE code = ?
	`, code)
	return scanAgent(row, false)
}

func (r *Registry) GetByName(name string) (*AgentRecord, error) {
	row := r.db.QueryRow(`
		SELECT uuid, code, hostname, COALESCE(name, ''), status, created_at, last_seen
		FROM agents WHERE name = ?
	`, name)
	return scanAgent(row, false)
}

// GetActiveByNameOrCode finds an active agent by assigned name or install code.
func (r *Registry) GetActiveByNameOrCode(identifier string) (*AgentRecord, error) {
	rec, err := r.GetByName(identifier)
	if err == nil {
		if rec.Status != protocol.StatusActive {
			return nil, fmt.Errorf("agent not active")
		}
		return rec, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	rec, err = r.GetByCode(identifier)
	if err != nil {
		return nil, err
	}
	if rec.Status != protocol.StatusActive {
		return nil, fmt.Errorf("agent not active")
	}
	return rec, nil
}

func (r *Registry) List() ([]AgentRecord, error) {
	rows, err := r.db.Query(`
		SELECT uuid, code, hostname, COALESCE(name, ''), status, created_at, last_seen
		FROM agents ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentRecord
	for rows.Next() {
		rec, err := scanAgent(rows, false)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func (r *Registry) AssignName(code, name string) (*AgentRecord, error) {
	var existing string
	err := r.db.QueryRow(`SELECT uuid FROM agents WHERE name = ?`, name).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("name %q already assigned", name)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	res, err := r.db.Exec(`
		UPDATE agents SET name = ?, status = ?
		WHERE code = ? AND status != ?
	`, name, protocol.StatusActive, code, protocol.StatusRevoked)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("agent code %q not found or revoked", code)
	}
	return r.GetByCode(code)
}

func (r *Registry) Rename(name, newName string) (*AgentRecord, error) {
	var existing string
	err := r.db.QueryRow(`SELECT uuid FROM agents WHERE name = ?`, newName).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("name %q already assigned", newName)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	res, err := r.db.Exec(`
		UPDATE agents SET name = ? WHERE name = ? AND status = ?
	`, newName, name, protocol.StatusActive)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("active agent %q not found", name)
	}
	return r.GetByName(newName)
}

func (r *Registry) Revoke(name string) error {
	res, err := r.db.Exec(`
		UPDATE agents SET status = ?, name = NULL WHERE name = ?
	`, protocol.StatusRevoked, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %q not found", name)
	}
	return nil
}

func (r *Registry) Delete(code string) error {
	res, err := r.db.Exec(`DELETE FROM agents WHERE code = ?`, code)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent code %q not found", code)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(row scanner, online bool) (*AgentRecord, error) {
	var rec AgentRecord
	var created, lastSeen string
	if err := row.Scan(&rec.UUID, &rec.Code, &rec.Hostname, &rec.Name, &rec.Status, &created, &lastSeen); err != nil {
		return nil, err
	}
	var err error
	rec.CreatedAt, err = time.Parse(time.RFC3339, created)
	if err != nil {
		return nil, err
	}
	rec.LastSeen, err = time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		return nil, err
	}
	rec.Online = online
	return &rec, nil
}
