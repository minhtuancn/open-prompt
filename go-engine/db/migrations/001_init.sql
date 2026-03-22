-- Users
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    display_name  TEXT,
    password_hash TEXT    NOT NULL,
    avatar_color  TEXT    NOT NULL DEFAULT '#6366f1',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login    DATETIME
);

-- Projects
CREATE TABLE IF NOT EXISTS projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    color      TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Prompts
CREATE TABLE IF NOT EXISTS prompts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    title      TEXT    NOT NULL,
    content    TEXT    NOT NULL,
    category   TEXT,
    tags       TEXT,
    is_slash   INTEGER NOT NULL DEFAULT 0,
    slash_name TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Skills
CREATE TABLE IF NOT EXISTS skills (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    prompt_id   INTEGER REFERENCES prompts(id) ON DELETE SET NULL,
    prompt_text TEXT,
    model       TEXT,
    provider    TEXT,
    config_json TEXT,
    tags        TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Settings per-user
CREATE TABLE IF NOT EXISTS settings (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key     TEXT    NOT NULL,
    value   TEXT,
    PRIMARY KEY (user_id, key)
);

-- History
CREATE TABLE IF NOT EXISTS history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER REFERENCES users(id) ON DELETE SET NULL,
    query         TEXT    NOT NULL,
    response      TEXT,
    provider      TEXT,
    model         TEXT,
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    latency_ms    INTEGER NOT NULL DEFAULT 0,
    status        TEXT    NOT NULL DEFAULT 'success',
    fallback_from TEXT,
    skill_id      INTEGER REFERENCES skills(id) ON DELETE SET NULL,
    timestamp     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Provider tokens (reference to system keychain)
CREATE TABLE IF NOT EXISTS provider_tokens (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id       TEXT    NOT NULL,
    auth_type         TEXT    NOT NULL,
    keychain_key      TEXT    NOT NULL,
    expires_at        DATETIME,
    refresh_token_key TEXT,
    detected_at       DATETIME,
    last_refreshed    DATETIME,
    is_active         INTEGER NOT NULL DEFAULT 1,
    UNIQUE(user_id, provider_id)
);

-- Model priority chain
CREATE TABLE IF NOT EXISTS model_priority (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    priority   INTEGER NOT NULL,
    provider   TEXT    NOT NULL,
    model      TEXT    NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    UNIQUE(user_id, priority)
);

-- Usage analytics daily aggregate
CREATE TABLE IF NOT EXISTS usage_daily (
    date           TEXT    NOT NULL,
    user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider       TEXT    NOT NULL,
    model          TEXT    NOT NULL,
    requests       INTEGER NOT NULL DEFAULT 0,
    input_tokens   INTEGER NOT NULL DEFAULT 0,
    output_tokens  INTEGER NOT NULL DEFAULT 0,
    errors         INTEGER NOT NULL DEFAULT 0,
    fallbacks      INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (date, user_id, provider, model)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_prompts_user ON prompts(user_id);
CREATE INDEX IF NOT EXISTS idx_prompts_slash ON prompts(slash_name) WHERE is_slash = 1;
CREATE INDEX IF NOT EXISTS idx_history_user_time ON history(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_history_cleanup ON history(timestamp);
