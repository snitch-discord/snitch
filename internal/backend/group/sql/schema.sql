CREATE TABLE IF NOT EXISTS users (
    user_id INTEGER PRIMARY KEY
) STRICT;

CREATE TABLE IF NOT EXISTS servers (
    server_id INTEGER PRIMARY KEY
) STRICT;

CREATE TABLE IF NOT EXISTS reports (
    report_id INTEGER PRIMARY KEY,
    report_text TEXT NOT NULL,
    reporter_id INTEGER NOT NULL REFERENCES users(user_id),
    reported_user_id INTEGER NOT NULL REFERENCES users(user_id),
    origin_server_id INTEGER NOT NULL REFERENCES servers(server_id)
) STRICT;


CREATE TABLE IF NOT EXISTS user_history (
    history_id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    global_name TEXT,
    changed_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
) STRICT;
