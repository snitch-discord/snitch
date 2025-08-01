package dbclient

import (
	"context"
	"fmt"
	"net/http"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
)

type Config struct {
	Host string
	Port string
}

type Client struct {
	dbClient snitchv1connect.DatabaseServiceClient
}

func New(config Config) *Client {
	baseURL := fmt.Sprintf("http://%s:%s", config.Host, config.Port)
	httpClient := &http.Client{}
	
	dbClient := snitchv1connect.NewDatabaseServiceClient(httpClient, baseURL)
	
	return &Client{
		dbClient: dbClient,
	}
}

// Metadata operations
func (c *Client) CreateGroup(ctx context.Context, groupID, groupName string) error {
	req := &snitchv1.CreateGroupRequest{
		GroupId:   groupID,
		GroupName: groupName,
	}
	
	_, err := c.dbClient.CreateGroup(ctx, connect.NewRequest(req))
	return err
}

func (c *Client) FindGroupByServer(ctx context.Context, serverID string) (string, error) {
	req := &snitchv1.FindGroupByServerRequest{
		ServerId: serverID,
	}
	
	resp, err := c.dbClient.FindGroupByServer(ctx, connect.NewRequest(req))
	if err != nil {
		return "", err
	}
	
	return resp.Msg.GroupId, nil
}

func (c *Client) AddServerToGroup(ctx context.Context, serverID, groupID string) error {
	req := &snitchv1.AddServerToGroupRequest{
		ServerId: serverID,
		GroupId:  groupID,
	}
	
	_, err := c.dbClient.AddServerToGroup(ctx, connect.NewRequest(req))
	return err
}

// Group database operations
func (c *Client) CreateGroupDatabase(ctx context.Context, groupID string) error {
	req := &snitchv1.CreateGroupDatabaseRequest{
		GroupId: groupID,
	}
	
	_, err := c.dbClient.CreateGroupDatabase(ctx, connect.NewRequest(req))
	return err
}

// Report operations
func (c *Client) CreateReport(ctx context.Context, groupID, userID, reporterID, serverID, reason string, evidenceURL *string) (int64, error) {
	req := &snitchv1.DbCreateReportRequest{
		GroupId:     groupID,
		UserId:      userID,
		ReporterId:  reporterID,
		ServerId:    serverID,
		Reason:      reason,
		EvidenceUrl: evidenceURL,
	}
	
	resp, err := c.dbClient.CreateReport(ctx, connect.NewRequest(req))
	if err != nil {
		return 0, err
	}
	
	return resp.Msg.ReportId, nil
}

func (c *Client) GetReport(ctx context.Context, groupID string, reportID int64) (*snitchv1.DbGetReportResponse, error) {
	req := &snitchv1.DbGetReportRequest{
		GroupId:  groupID,
		ReportId: reportID,
	}
	
	resp, err := c.dbClient.GetReport(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	
	return resp.Msg, nil
}

func (c *Client) ListReports(ctx context.Context, groupID string, userID *string, limit, offset *int32) ([]*snitchv1.DbGetReportResponse, error) {
	req := &snitchv1.DbListReportsRequest{
		GroupId: groupID,
		UserId:  userID,
		Limit:   limit,
		Offset:  offset,
	}
	
	resp, err := c.dbClient.ListReports(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	
	return resp.Msg.Reports, nil
}

func (c *Client) DeleteReport(ctx context.Context, groupID string, reportID int64) error {
	req := &snitchv1.DbDeleteReportRequest{
		GroupId:  groupID,
		ReportId: reportID,
	}
	
	_, err := c.dbClient.DeleteReport(ctx, connect.NewRequest(req))
	return err
}

// User history operations
func (c *Client) CreateUserHistory(ctx context.Context, groupID, userID, serverID, action string, reason, evidenceURL *string) (int64, error) {
	req := &snitchv1.DbCreateUserHistoryRequest{
		GroupId:     groupID,
		UserId:      userID,
		ServerId:    serverID,
		Action:      action,
		Reason:      reason,
		EvidenceUrl: evidenceURL,
	}
	
	resp, err := c.dbClient.CreateUserHistory(ctx, connect.NewRequest(req))
	if err != nil {
		return 0, err
	}
	
	return resp.Msg.HistoryId, nil
}

func (c *Client) GetUserHistory(ctx context.Context, groupID, userID string, limit, offset *int32) ([]*snitchv1.DbUserHistoryEntry, error) {
	req := &snitchv1.DbGetUserHistoryRequest{
		GroupId: groupID,
		UserId:  userID,
		Limit:   limit,
		Offset:  offset,
	}
	
	resp, err := c.dbClient.GetUserHistory(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	
	return resp.Msg.Entries, nil
}

// Server operations
func (c *Client) ListServers(ctx context.Context, groupID string) ([]*snitchv1.ServerEntry, error) {
	req := &snitchv1.ListServersRequest{
		GroupId: groupID,
	}
	
	resp, err := c.dbClient.ListServers(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	
	return resp.Msg.Servers, nil
}