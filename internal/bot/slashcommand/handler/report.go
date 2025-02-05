package handler

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
)

func handleNewReport(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.ReportServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	reporterID, err := strconv.Atoi(interaction.Member.User.ID)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed to convert reporter ID", "Error", err)
		return
	}

	reportedUserOption, ok := optionMap["reported-user"]
	if !ok {
		slogger.ErrorContext(ctx, "Failed to get reported user option", "Error", err)
		return
	}

	reportedUser := reportedUserOption.UserValue(session)
	reportedID, err := strconv.Atoi(reportedUser.ID)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed to convert reported ID", "Error", err)
		return
	}

	reportReason := ""
	reportReasonOption, ok := optionMap["report-reason"]
	if ok {
		reportReason = reportReasonOption.StringValue()
	}

	reportRequest := connect.NewRequest(&snitchv1.CreateReportRequest{ReportText: reportReason, ReporterId: int32(reporterID), ReportedId: int32(reportedID)})
	reportRequest.Header().Add("X-Server-ID", interaction.GuildID)
	reportResponse, err := client.CreateReport(ctx, reportRequest)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		if err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Couldn't report user, error: %s", err.Error()),
			},
		}); err != nil {
			slogger.ErrorContext(ctx, "Couldn't Write Discord Response", "Error", err)
		}
		return
	}

	messageContent := fmt.Sprintf("Reported user: %s; Report reason: %s; Report ID: %d", reportedUser.Username, reportReason, reportResponse.Msg.ReportId)
	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messageContent,
		},
	}); err != nil {
		slogger.ErrorContext(ctx, "Failed to respond", "Error", err)
	}
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

	var reporterUserID *int32
	reporterUserOption, ok := optionMap["reporter-user"]
	if ok {
		res, err := strconv.Atoi(reporterUserOption.UserValue(session).ID)
		if err == nil {
			final := int32(res)
			reporterUserID = &final
		}
	}

	var reportedUserID *int32
	reportedUserOption, ok := optionMap["reported-user"]
	if ok {
		res, err := strconv.Atoi(reportedUserOption.UserValue(session).ID)
		if err == nil {
			final := int32(res)
			reportedUserID = &final
		}
	}

	slogger.InfoContext(ctx, "List Params", "Reporter", reporterUserID, "Reported", reportedUserID)

	listReportRequest := connect.NewRequest(&snitchv1.ListReportsRequest{ReporterId: reporterUserID, ReportedId: reportedUserID})
	listReportRequest.Header().Add("X-Server-ID", interaction.GuildID)
	listReportResponse, err := client.ListReports(ctx, listReportRequest)

	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		if err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Couldn't list reports, error: %s", err.Error()),
			},
		}); err != nil {
			slogger.ErrorContext(ctx, "Couldn't Write Discord Response", "Error", err)
		}
		return
	}

	var responseStringBuilder strings.Builder
	reports := listReportResponse.Msg.Reports
	for index, report := range reports {
		responseStringBuilder.WriteString(fmt.Sprintf("Report %d: %s\n", index, report))
	}

	var messageContent string
	if responseStringBuilder.Len() == 0 {
		messageContent = "No reports found!"
	} else {
		messageContent = responseStringBuilder.String()
	}

	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messageContent,
		},
	}); err != nil {
		slogger.ErrorContext(ctx, "Failed to respond", "Error", err)
	}
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

	reportIDOption, ok := optionMap["report-id"]
	if !ok {
		slogger.ErrorContext(ctx, "Failed to get reported user option")
		return
	}
	reportID := reportIDOption.IntValue()

	deleteReportRequest := connect.NewRequest(&snitchv1.DeleteReportRequest{ReportId: int32(reportID)})
	deleteReportRequest.Header().Add("X-Server-ID", interaction.GuildID)
	deleteReportResponse, err := client.DeleteReport(ctx, deleteReportRequest)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		if err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Couldn't delete report, error: %s", err.Error()),
			},
		}); err != nil {
			slogger.ErrorContext(ctx, "Couldn't Write Discord Response", "Error", err)
		}
		return
	}

	messageContent := fmt.Sprintf("Deleted report %d", deleteReportResponse.Msg.ReportId)

	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messageContent,
		},
	}); err != nil {
		slogger.ErrorContext(ctx, "Failed to respond", "Error", err)
	}
}

func CreateReportCommandHandler(botconfig botconfig.BotConfig, httpClient http.Client) slashcommand.SlashCommandHandlerFunc {
	backendURL, err := botconfig.BackendURL()
	if err != nil {
		log.Fatal(backendURL)
	}
	reportServiceClient := snitchv1connect.NewReportServiceClient(&httpClient, backendURL.String())

	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		options := interaction.ApplicationCommandData().Options

		switch options[0].Name {
		case "new":
			handleNewReport(ctx, session, interaction, reportServiceClient)
		case "list":
			handleListReports(ctx, session, interaction, reportServiceClient)
		case "delete":
			handleDeleteReport(ctx, session, interaction, reportServiceClient)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
