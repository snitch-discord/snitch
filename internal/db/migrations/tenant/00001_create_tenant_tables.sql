-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY
) STRICT;

CREATE TABLE IF NOT EXISTS servers (
    server_id TEXT PRIMARY KEY
) STRICT;

CREATE TABLE IF NOT EXISTS reports (
    report_id INTEGER PRIMARY KEY,
    report_text TEXT NOT NULL CHECK(length(report_text) <= 2000 AND length(report_text) > 0),
    reporter_id TEXT NOT NULL REFERENCES users(user_id),
    reported_user_id TEXT NOT NULL REFERENCES users(user_id),
    origin_server_id TEXT NOT NULL REFERENCES servers(server_id),
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
) STRICT;

CREATE TABLE IF NOT EXISTS user_history (
    history_id INTEGER PRIMARY KEY,
    user_id TEXT NOT NULL,
    server_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK(length(action) <= 100),
    reason TEXT CHECK(reason IS NULL OR length(reason) <= 1000),
    evidence_url TEXT CHECK(evidence_url IS NULL OR length(evidence_url) <= 500),
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (server_id) REFERENCES servers(server_id)
) STRICT;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_reports_reporter_id ON reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_reports_reported_user_id ON reports(reported_user_id);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);
CREATE INDEX IF NOT EXISTS idx_reports_origin_server_id ON reports(origin_server_id);
CREATE INDEX IF NOT EXISTS idx_user_history_user_id ON user_history(user_id);
CREATE INDEX IF NOT EXISTS idx_user_history_server_id ON user_history(server_id);
CREATE INDEX IF NOT EXISTS idx_user_history_created_at ON user_history(created_at);
CREATE INDEX IF NOT EXISTS idx_reports_user_date ON reports(reported_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_reports_server_date ON reports(origin_server_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_reports_server_date;
DROP INDEX IF EXISTS idx_reports_user_date;
DROP INDEX IF EXISTS idx_user_history_created_at;
DROP INDEX IF EXISTS idx_user_history_server_id;
DROP INDEX IF EXISTS idx_user_history_user_id;
DROP INDEX IF EXISTS idx_reports_origin_server_id;
DROP INDEX IF EXISTS idx_reports_created_at;
DROP INDEX IF EXISTS idx_reports_reported_user_id;
DROP INDEX IF EXISTS idx_reports_reporter_id;
DROP TABLE IF EXISTS user_history;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS servers;
DROP TABLE IF EXISTS users;