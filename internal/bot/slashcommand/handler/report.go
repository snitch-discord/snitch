package handler

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/messageutil"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/bot/transport"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
	"time"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
)

func handleNewReport(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.ReportServiceClient, userClient snitchv1connect.UserHistoryServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	reporterID := interaction.Member.User.ID

	reportedUserOption, ok := optionMap["reported-user"]

	if !ok {
		slogger.ErrorContext(ctx, "Failed to get reported user option", "Error", ok)
	}

	reportedUser := reportedUserOption.UserValue(session)
	reportedID := reportedUser.ID

	reportReason := ""
	reportReasonOption, ok := optionMap["report-reason"]
	if ok {
		reportReason = reportReasonOption.StringValue()
	}

	reportRequest := connect.NewRequest(&snitchv1.CreateReportRequest{ReportText: reportReason, ReporterId: reporterID, ReportedId: reportedUser.ID})
	reportResponse, err := client.CreateReport(ctx, reportRequest)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't report user, error: %s", err.Error()))
		return
	}

	userRequest := connect.NewRequest(&snitchv1.CreateUserHistoryRequest{UserId: reportedID, Username: reportedUser.Username, GlobalName: reportedUser.GlobalName, ChangedAt: time.Now().UTC().Format(time.RFC3339)})
	_, err = userClient.CreateUserHistory(ctx, userRequest)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't report user, error: %s", err.Error()))
		return
	}

	messageContent := fmt.Sprintf("Reported user: %s; Report reason: %s; Report ID: %d", reportedUser.Username, reportReason, reportResponse.Msg.ReportId)
	messageutil.SimpleRespondContext(ctx, session, interaction, messageContent)
}

func handleListReports(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.ReportServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var reporterUserID *string
	reporterUserOption, ok := optionMap["reporter-user"]
	if ok {
		reporterUserID = &reporterUserOption.UserValue(session).ID
	}

	var reportedUserID *string
	reportedUserOption, ok := optionMap["reported-user"]
	if ok {
		reportedUserID = &reportedUserOption.UserValue(session).ID

	}

	slogger.InfoContext(ctx, "List Params", "Reporter", reporterUserID, "Reported", reportedUserID)

	listReportRequest := connect.NewRequest(&snitchv1.ListReportsRequest{ReporterId: reporterUserID, ReportedId: reportedUserID})
	listReportResponse, err := client.ListReports(ctx, listReportRequest)

	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't list reports, error: %s", err.Error()))
		return
	}

	reportEmbed := messageutil.NewEmbed().
		SetTitle("Reports").
		SetDescription("Report List")

	reports := listReportResponse.Msg.Reports
	for index, report := range reports {
		headerField := fmt.Sprintf("%d: Reporter ID: %s, Reported ID: %s", index, report.ReporterId, report.ReportedId)
		reportEmbed.AddField(headerField, report.ReportText)
	}

	messageutil.EmbedRespondContext(ctx, session, interaction, []*discordgo.MessageEmbed{reportEmbed.MessageEmbed})
}

func handleDeleteReport(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.ReportServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var reportID int64
	reportIDOption, ok := optionMap["report-id"]
	if ok {
		reportID = reportIDOption.IntValue()
	}

	deleteReportRequest := connect.NewRequest(&snitchv1.DeleteReportRequest{ReportId: reportID})
	deleteReportResponse, err := client.DeleteReport(ctx, deleteReportRequest)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't delete report, error: %s", err.Error()))
		return
	}

	messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Deleted report %d", deleteReportResponse.Msg.ReportId))
}

func CreateReportCommandHandler(botconfig botconfig.BotConfig, httpClient http.Client) slashcommand.SlashCommandHandlerFunc {
	backendURL, err := botconfig.BackendURL()
	if err != nil {
		log.Fatal(backendURL)
	}
	reportServiceClient := snitchv1connect.NewReportServiceClient(&httpClient, backendURL.String())
	userServiceClient := snitchv1connect.NewUserHistoryServiceClient(&httpClient, backendURL.String())
	registrarServiceClient := snitchv1connect.NewRegistrarServiceClient(&httpClient, backendURL.String())

	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		getGroupReq := connect.NewRequest(&snitchv1.GetGroupForServerRequest{ServerId: interaction.GuildID})
		getGroupResp, err := registrarServiceClient.GetGroupForServer(ctx, getGroupReq)
		if err != nil {
			slogger.ErrorContext(ctx, "failed to get group for server", "error", err)
			messageutil.SimpleRespondContext(ctx, session, interaction, "Failed to get group for server.")
			return
		}

		ctx = transport.WithAuthInfo(ctx, interaction.GuildID, getGroupResp.Msg.GroupId)

		options := interaction.ApplicationCommandData().Options

		switch options[0].Name {
		case "new":
			handleNewReport(ctx, session, interaction, reportServiceClient, userServiceClient)
		case "list":
			handleListReports(ctx, session, interaction, reportServiceClient)
		case "delete":
			handleDeleteReport(ctx, session, interaction, reportServiceClient)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
