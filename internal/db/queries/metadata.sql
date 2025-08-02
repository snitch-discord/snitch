-- Metadata database queries (groups and servers)

-- name: CreateGroup :exec
INSERT INTO groups (group_id, group_name) VALUES (?, ?);

-- name: FindGroupByServer :one
SELECT group_id FROM servers WHERE server_id = ?;

-- name: AddServerToGroup :exec
INSERT INTO servers (server_id, output_channel, group_id, permission_level) VALUES (?, ?, ?, ?);