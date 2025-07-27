-- name: FindGroupIDByServerID :one
SELECT group_id 
FROM servers 
WHERE server_id = ?;

-- name: AddServerToGroup :exec
INSERT INTO servers (
    server_id, 
    output_channel, 
    group_id, 
    permission_level
) VALUES (
    ?, 
    69420, 
    ?, 
    777
);

-- name: InsertGroup :exec
INSERT INTO groups (
    group_id, 
    group_name
) VALUES (
    ?,
    ?
);

-- name: CreateGroupTable :exec
CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY,
    group_name TEXT NOT NULL
) STRICT;

-- name: CreateServerTable :exec
CREATE TABLE IF NOT EXISTS servers (
    server_id TEXT NOT NULL,
    output_channel INTEGER NOT NULL,
    group_id TEXT NOT NULL REFERENCES groups(group_id),
    permission_level INTEGER NOT NULL,
    PRIMARY KEY (server_id, group_id)
) STRICT;


