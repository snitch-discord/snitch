package handler

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/shared/ctxutil"

	"github.com/bwmarrin/discordgo"
)

func handleNewReport(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
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

	responseContent := fmt.Sprintf("Reported user: %s; Report reason: %s", reportedUser.Username, reportReason)

	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
		},
	}); err != nil {
		slogger.ErrorContext(ctx, "Failed to respond", "Error", err)
	}
}

func handleListReports(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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

func handleDeleteReport(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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

func CreateReportCommandHandler(botconfig botconfig.BotConfig) slashcommand.SlashCommandHandlerFunc {
	backendURL, err := botconfig.BackendURL()
	if err != nil {
		log.Fatal(backendURL)
	}

	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		options := interaction.ApplicationCommandData().Options

		switch options[0].Name {
		case "new":
			handleNewReport(ctx, session, interaction)
		case "list":
			handleListReports(ctx, session, interaction)
		case "delete":
			handleDeleteReport(ctx, session, interaction)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
