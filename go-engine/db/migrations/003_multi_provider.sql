-- Migration 003: Phase 2A Multi-Provider

-- Custom alias mapping per user
CREATE TABLE IF NOT EXISTS model_aliases (
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alias       TEXT    NOT NULL,
    provider_id TEXT    NOT NULL,
    PRIMARY KEY (user_id, alias)
);

-- Custom gateways (OpenAI-compat servers)
CREATE TABLE IF NOT EXISTS custom_gateways (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    display_name TEXT    NOT NULL,
    base_url     TEXT    NOT NULL,
    api_key      TEXT    DEFAULT '',
    default_model TEXT   DEFAULT '',
    aliases      TEXT    DEFAULT '[]',
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_custom_gateways_user ON custom_gateways(user_id);
