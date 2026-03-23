package repos

import (
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Plugin là một plugin đã cài
type Plugin struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Type       string `json:"type"`
	ConfigJSON string `json:"config_json"`
	Enabled    bool   `json:"enabled"`
	Source     string `json:"source"`
	SourcePath string `json:"source_path"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// PluginRepo xử lý CRUD cho plugins
type PluginRepo struct {
	db *db.DB
}

// NewPluginRepo tạo repo mới
func NewPluginRepo(database *db.DB) *PluginRepo {
	return &PluginRepo{db: database}
}

// Install cài plugin mới
func (r *PluginRepo) Install(userID int64, name, version, pluginType, configJSON, source, sourcePath string) (int64, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := r.db.Exec(
		`INSERT OR REPLACE INTO plugins (user_id, name, version, type, config_json, enabled, source, source_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
		userID, name, version, pluginType, configJSON, source, sourcePath, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("install plugin: %w", err)
	}
	return result.LastInsertId()
}

// List trả về danh sách plugins của user
func (r *PluginRepo) List(userID int64) ([]Plugin, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, version, type, config_json, enabled, COALESCE(source,''), COALESCE(source_path,''), created_at, updated_at
		 FROM plugins WHERE user_id = ? ORDER BY name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	var result []Plugin
	for rows.Next() {
		var p Plugin
		var enabled int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Version, &p.Type, &p.ConfigJSON, &enabled, &p.Source, &p.SourcePath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		result = append(result, p)
	}
	return result, rows.Err()
}

// ListByType trả về plugins theo type
func (r *PluginRepo) ListByType(userID int64, pluginType string) ([]Plugin, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, name, version, type, config_json, enabled, COALESCE(source,''), COALESCE(source_path,''), created_at, updated_at
		 FROM plugins WHERE user_id = ? AND type = ? AND enabled = 1 ORDER BY name`,
		userID, pluginType,
	)
	if err != nil {
		return nil, fmt.Errorf("list plugins by type: %w", err)
	}
	defer rows.Close()

	var result []Plugin
	for rows.Next() {
		var p Plugin
		var enabled int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Version, &p.Type, &p.ConfigJSON, &enabled, &p.Source, &p.SourcePath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		result = append(result, p)
	}
	return result, rows.Err()
}

// Toggle bật/tắt plugin
func (r *PluginRepo) Toggle(id, userID int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := r.db.Exec(`UPDATE plugins SET enabled = ?, updated_at = datetime('now') WHERE id = ? AND user_id = ?`, val, id, userID)
	return err
}

// Uninstall xóa plugin
func (r *PluginRepo) Uninstall(id, userID int64) error {
	_, err := r.db.Exec(`DELETE FROM plugins WHERE id = ? AND user_id = ?`, id, userID)
	return err
}
