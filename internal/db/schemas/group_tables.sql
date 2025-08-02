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