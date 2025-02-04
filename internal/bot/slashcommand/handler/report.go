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

	reporterID := interaction.Member.User.ID
	intReporterID, err := strconv.Atoi(reporterID)
	if err != nil {
		log.Print(err)
		return
	}

	reportedUserOption, ok := optionMap["reported-user"]
	if !ok {
		slogger.ErrorContext(ctx, "Failed to get reported user option")

		return
	}
	reportedUser := reportedUserOption.UserValue(session)

	reportReason := ""
	reportReasonOption, ok := optionMap["report-reason"]
	if ok {
		reportReason = reportReasonOption.StringValue()
	}

	intReportedID, err := strconv.Atoi(reportedUser.ID)
	if err != nil {
		log.Print(err)
		return
	}

	reportRequest := connect.NewRequest(&snitchv1.CreateReportRequest{ReportText: reportReason, ReporterId: int32(intReporterID), ReportedId: int32(intReportedID)})
	reportRequest.Header().Add("X-Server-ID", interaction.GuildID)
	reportResponse, err := client.CreateReport(ctx, reportRequest)
	responseContent := fmt.Sprintf("Reported user: %s; Report reason: %s; Report ID: %d", reportedUser.Username, reportReason, reportResponse.Msg.ReportId)

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

	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
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

	reportedUserName := ""
	reportedUserOption, ok := optionMap["reported-user"]
	if ok {
		reportedUserName = reportedUserOption.UserValue(session).Username
	}

	reporterUserName := ""
	reporterUserOption, ok := optionMap["reporter-user"]
	if ok {
		reporterUserName = reporterUserOption.UserValue(session).Username
	}

	responseContent := fmt.Sprintf("Reported user: %s; Reporter user: %s", reportedUserName, reporterUserName)

	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
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

	responseContent := fmt.Sprintf("Delete report %s", reportIDOption.StringValue())

	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
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
