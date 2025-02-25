// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package sqlc

import (
	"github.com/google/uuid"
)

type Group struct {
	GroupID   uuid.UUID `json:"group_id"`
	GroupName string    `json:"group_name"`
}

type Server struct {
	ServerID        interface{} `json:"server_id"`
	OutputChannel   int         `json:"output_channel"`
	GroupID         uuid.UUID   `json:"group_id"`
	PermissionLevel int         `json:"permission_level"`
}
