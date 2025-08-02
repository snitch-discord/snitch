CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY,
    group_name TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS servers (
    server_id TEXT NOT NULL,
    output_channel INTEGER NOT NULL,
    group_id TEXT NOT NULL REFERENCES groups(group_id),
    permission_level INTEGER NOT NULL,
    PRIMARY KEY (server_id, group_id)
) STRICT;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_servers_group_id ON servers(group_id);