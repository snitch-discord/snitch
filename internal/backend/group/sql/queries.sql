-- name: GetAllReports :many
SELECT 
    report_text,
    reporter_id,
    reported_user_id,
    origin_server_id
FROM reports;

-- name: AddServer :exec
INSERT INTO servers (
    server_id
) VALUES (?);

-- name: AddUser :exec
INSERT OR IGNORE INTO users (
    user_id
) VALUES (?);

-- name: CreateReport :one
INSERT INTO reports (
    report_text,
    reporter_id, 
    reported_user_id,
    origin_server_id
) values (?, ?, ?, ?)
RETURNING report_id;

-- name: DeleteReport :one
DELETE FROM reports
WHERE report_id = ?
RETURNING report_id;

-- name: CreateUserTable :exec
CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY
) STRICT;

-- name: CreateServerTable :exec
CREATE TABLE IF NOT EXISTS servers (
    server_id TEXT PRIMARY KEY
) STRICT;

-- name: CreateReportTable :exec
CREATE TABLE IF NOT EXISTS reports (
    report_id TEXT PRIMARY KEY,
    report_text TEXT NOT NULL,
    reporter_id TEXT NOT NULL REFERENCES users(user_id),
    reported_user_id TEXT NOT NULL REFERENCES users(user_id),
    origin_server_id TEXT NOT NULL REFERENCES servers(server_id)
) STRICT;


-- name: CreateUserHistoryTable :exec
CREATE TABLE IF NOT EXISTS user_history (
    history_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL,
    global_name TEXT,
    changed_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
) STRICT;

-- name: CreateUserHistory :one
INSERT INTO user_history (
    history_id,
    user_id,
    username,
    global_name,
    changed_at
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserHistory :many
SELECT * FROM user_history
WHERE user_id = ?
ORDER BY changed_at DESC;

-- name: GetUser :one
SELECT * FROM users
WHERE user_id = ?;

