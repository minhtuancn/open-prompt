-- Shared prompts cho marketplace
CREATE TABLE IF NOT EXISTS shared_prompts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    content     TEXT    NOT NULL,
    description TEXT,
    category    TEXT,
    tags        TEXT,
    downloads   INTEGER NOT NULL DEFAULT 0,
    is_public   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shared_prompts_category ON shared_prompts(category);
CREATE INDEX IF NOT EXISTS idx_shared_prompts_downloads ON shared_prompts(downloads DESC);
