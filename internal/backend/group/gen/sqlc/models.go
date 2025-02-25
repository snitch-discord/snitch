// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package sqlc

type Report struct {
	ReportID       string `json:"report_id"`
	ReportText     string `json:"report_text"`
	ReporterID     string `json:"reporter_id"`
	ReportedUserID string `json:"reported_user_id"`
	OriginServerID string `json:"origin_server_id"`
}

type Server struct {
	ServerID string `json:"server_id"`
}

type User struct {
	UserID string `json:"user_id"`
}

type UserHistory struct {
	HistoryID  string `json:"history_id"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	ChangedAt  string `json:"changed_at"`
}
