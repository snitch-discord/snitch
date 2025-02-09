// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package sqlc

import (
	"context"

	"github.com/google/uuid"
)

const addServerToGroup = `-- name: AddServerToGroup :exec
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
)
`

type AddServerToGroupParams struct {
	ServerID interface{} `json:"server_id"`
	GroupID  uuid.UUID   `json:"group_id"`
}

func (q *Queries) AddServerToGroup(ctx context.Context, arg AddServerToGroupParams) error {
	_, err := q.exec(ctx, q.addServerToGroupStmt, addServerToGroup, arg.ServerID, arg.GroupID)
	return err
}

const createGroupTable = `-- name: CreateGroupTable :exec
CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY,
    group_name TEXT NOT NULL
) STRICT
`

func (q *Queries) CreateGroupTable(ctx context.Context) error {
	_, err := q.exec(ctx, q.createGroupTableStmt, createGroupTable)
	return err
}

const createServerTable = `-- name: CreateServerTable :exec
CREATE TABLE IF NOT EXISTS servers (
    server_id INTEGER NOT NULL,
    output_channel INTEGER NOT NULL,
    group_id TEXT NOT NULL REFERENCES groups(group_id),
    permission_level INTEGER NOT NULL,
    PRIMARY KEY (server_id, group_id)
) STRICT
`

func (q *Queries) CreateServerTable(ctx context.Context) error {
	_, err := q.exec(ctx, q.createServerTableStmt, createServerTable)
	return err
}

const findGroupIDByServerID = `-- name: FindGroupIDByServerID :one
SELECT group_id 
FROM servers 
WHERE server_id = ?
`

func (q *Queries) FindGroupIDByServerID(ctx context.Context, serverID interface{}) (uuid.UUID, error) {
	row := q.queryRow(ctx, q.findGroupIDByServerIDStmt, findGroupIDByServerID, serverID)
	var group_id uuid.UUID
	err := row.Scan(&group_id)
	return group_id, err
}

const insertGroup = `-- name: InsertGroup :exec
INSERT INTO groups (
    group_id, 
    group_name
) VALUES (
    ?,
    ?
)
`

type InsertGroupParams struct {
	GroupID   uuid.UUID `json:"group_id"`
	GroupName string    `json:"group_name"`
}

func (q *Queries) InsertGroup(ctx context.Context, arg InsertGroupParams) error {
	_, err := q.exec(ctx, q.insertGroupStmt, insertGroup, arg.GroupID, arg.GroupName)
	return err
}
