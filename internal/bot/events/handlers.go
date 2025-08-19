package events

import (
	"fmt"
	"log/slog"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"github.com/bwmarrin/discordgo"
)

func CreateReportCreatedHandler(logger *slog.Logger) EventHandler {
	return func(session *discordgo.Session, event *snitchv1.Event) error {
		reportCreated := event.GetReportCreated()
		if reportCreated == nil {
			return fmt.Errorf("expected report created event data")
		}

		logger.Info("Report created event received",
			"report_id", reportCreated.ReportId,
			"reporter_id", reportCreated.ReporterId,
			"reported_id", reportCreated.ReportedId,
			"server_id", event.ServerId,
		)

		return nil
	}
}

func CreateReportDeletedHandler(logger *slog.Logger) EventHandler {
	return func(session *discordgo.Session, event *snitchv1.Event) error {
		reportDeleted := event.GetReportDeleted()
		if reportDeleted == nil {
			return fmt.Errorf("expected report deleted event data")
		}

		logger.Info("Report deleted event received",
			"report_id", reportDeleted.ReportId,
			"server_id", event.ServerId,
		)

		return nil
	}
}

func CreateUserBannedHandler(logger *slog.Logger) EventHandler {
	return func(session *discordgo.Session, event *snitchv1.Event) error {
		userBanned := event.GetUserBanned()
		if userBanned == nil {
			return fmt.Errorf("expected user banned event data")
		}

		logger.Info("User banned event received",
			"user_id", userBanned.UserId,
			"server_id", userBanned.ServerId,
			"reason", userBanned.Reason,
		)

		return nil
	}
}
