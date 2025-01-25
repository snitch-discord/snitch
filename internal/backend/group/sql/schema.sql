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

