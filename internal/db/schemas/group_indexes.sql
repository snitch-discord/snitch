CREATE INDEX IF NOT EXISTS idx_reports_reporter_id ON reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_reports_reported_user_id ON reports(reported_user_id);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);
CREATE INDEX IF NOT EXISTS idx_reports_origin_server_id ON reports(origin_server_id);
CREATE INDEX IF NOT EXISTS idx_user_history_user_id ON user_history(user_id);
CREATE INDEX IF NOT EXISTS idx_user_history_server_id ON user_history(server_id);
CREATE INDEX IF NOT EXISTS idx_user_history_created_at ON user_history(created_at);
CREATE INDEX IF NOT EXISTS idx_reports_user_date ON reports(reported_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_reports_server_date ON reports(origin_server_id, created_at);