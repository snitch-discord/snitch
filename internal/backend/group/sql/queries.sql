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
    user_id INTEGER PRIMARY KEY
) STRICT;

-- name: CreateServerTable :exec
CREATE TABLE IF NOT EXISTS servers (
    server_id INTEGER PRIMARY KEY
) STRICT;

-- name: CreateReportTable :exec
CREATE TABLE IF NOT EXISTS reports (
    report_id INTEGER PRIMARY KEY,
    report_text TEXT NOT NULL,
    reporter_id INTEGER NOT NULL REFERENCES users(user_id),
    reported_user_id INTEGER NOT NULL REFERENCES users(user_id),
    origin_server_id INTEGER NOT NULL REFERENCES servers(server_id)
) STRICT;


