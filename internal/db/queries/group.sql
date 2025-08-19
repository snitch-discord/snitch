-- Group database queries (reports and users)

-- name: EnsureUserExists :exec
INSERT OR IGNORE INTO users (user_id) VALUES (?);

-- name: EnsureServerExists :exec
INSERT OR IGNORE INTO servers (server_id) VALUES (?);

-- name: CreateReport :one
INSERT INTO reports (report_text, reporter_id, reported_user_id, origin_server_id) 
VALUES (?, ?, ?, ?) RETURNING report_id;

-- name: GetReport :one
SELECT report_id, report_text, reporter_id, reported_user_id, origin_server_id, created_at 
FROM reports WHERE report_id = ?;

-- name: ListReports :many
SELECT report_id, report_text, reporter_id, reported_user_id, origin_server_id, created_at 
FROM reports
ORDER BY created_at DESC;

-- name: ListReportsByUser :many
SELECT report_id, report_text, reporter_id, reported_user_id, origin_server_id, created_at 
FROM reports 
WHERE reported_user_id = ?
ORDER BY created_at DESC;

-- name: DeleteReport :execrows
DELETE FROM reports WHERE report_id = ?;

-- User history queries
-- name: CreateUserHistory :one
INSERT INTO user_history (user_id, server_id, action, reason, evidence_url) 
VALUES (?, ?, ?, ?, ?) RETURNING history_id;

-- name: GetUserHistory :many
SELECT history_id, user_id, server_id, action, reason, evidence_url, created_at 
FROM user_history 
WHERE user_id = ? 
ORDER BY created_at DESC;