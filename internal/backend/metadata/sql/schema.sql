CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY,
    group_name TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS servers (
    server_id INTEGER NOT NULL,
    output_channel INTEGER NOT NULL,
    group_id TEXT NOT NULL REFERENCES groups(group_id),
    permission_level INTEGER NOT NULL,
    PRIMARY KEY (server_id, group_id)
) STRICT;

