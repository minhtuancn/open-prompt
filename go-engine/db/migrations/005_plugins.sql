-- Migration 005: Plugin system

CREATE TABLE IF NOT EXISTS plugins (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    version     TEXT    NOT NULL DEFAULT '1.0.0',
    type        TEXT    NOT NULL, -- 'provider' | 'skill' | 'formatter'
    config_json TEXT    NOT NULL DEFAULT '{}',
    enabled     INTEGER NOT NULL DEFAULT 1,
    source      TEXT,             -- 'local' | 'url' | 'builtin'
    source_path TEXT,             -- file path hoặc URL
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_plugins_user ON plugins(user_id);
CREATE INDEX IF NOT EXISTS idx_plugins_type ON plugins(user_id, type);
