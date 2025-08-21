package handler

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"snitch/internal/bot/auth"
	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/messageutil"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
)

func handleUserHistory(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.UserHistoryServiceClient, tokenGenerator *auth.TokenGenerator) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	token, err := tokenGenerator.Generate(interaction.GuildID)
	if err != nil {
		slogger.ErrorContext(ctx, "failed to generate token", "error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, "Failed to generate token.")
		return
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	reporterID, err := strconv.Atoi(interaction.Member.User.ID)
	slogger.InfoContext(ctx, "Reporter ID", "ID", reporterID)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed to convert reporter ID", "Error", err)
		return
	}

	reportedUserOption, ok := optionMap["user-id"]
	if !ok {
		slogger.ErrorContext(ctx, "Failed to get user id option", "Error", err)
		return
	}

	reportedUser := reportedUserOption.UserValue(session)

	reportRequest := connect.NewRequest(&snitchv1.CreateUserHistoryRequest{UserId: reportedUser.ID, Username: reportedUser.Username, ChangedAt: time.Now().UTC().Format(time.RFC3339)})
	reportRequest.Header().Add("Authorization", "Bearer "+token)
	reportResponse, err := client.CreateUserHistory(ctx, reportRequest)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't create user history, error: %s", err.Error()))
		return
	}

	messageContent := fmt.Sprintf("User history for: %s; User ID: %s", reportedUser.Username, reportResponse.Msg.UserId)
	messageutil.SimpleRespondContext(ctx, session, interaction, messageContent)
}

func handleListUserHistory(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.UserHistoryServiceClient, tokenGenerator *auth.TokenGenerator) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	token, err := tokenGenerator.Generate(interaction.GuildID)
	if err != nil {
		slogger.ErrorContext(ctx, "failed to generate token", "error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, "Failed to generate token.")
		return
	}

	options := interaction.ApplicationCommandData().Options[0].Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	userIDOption, ok := optionMap["user"]
	if !ok {
		messageutil.SimpleRespondContext(ctx, session, interaction, "Missing user-id option")
		return
	}

	user := userIDOption.UserValue(session)
	if user == nil {
		messageutil.SimpleRespondContext(ctx, session, interaction, "Could not resolve user")
		return
	}

	userID := user.ID

	listUserHistoryRequest := connect.NewRequest(&snitchv1.ListUserHistoryRequest{UserId: userID})
	listUserHistoryRequest.Header().Add("Authorization", "Bearer "+token)
	listUserHistoryResponse, err := client.ListUserHistory(ctx, listUserHistoryRequest)

	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't list user history, error: %s", err.Error()))
		return
	}

	messageHistoryEmbed := messageutil.NewEmbed().
		SetTitle("User History")

	history := listUserHistoryResponse.Msg.UserHistory
	for index, h := range history {
		messageHistoryEmbed.AddField(fmt.Sprintf("History %d", index), h.Username)
	}

	messageutil.EmbedRespondContext(ctx, session, interaction, []*discordgo.MessageEmbed{messageHistoryEmbed.MessageEmbed})
}

func CreateUserCommandHandler(botconfig botconfig.BotConfig, httpClient http.Client, tokenGenerator *auth.TokenGenerator) slashcommand.SlashCommandHandlerFunc {
	backendURL, err := botconfig.BackendURL()
	if err != nil {
		log.Fatal(backendURL)
	}
	userHistoryServiceClient := snitchv1connect.NewUserHistoryServiceClient(&httpClient, backendURL.String())

	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		options := interaction.ApplicationCommandData().Options

		switch options[0].Name {
		case "new":
			handleUserHistory(ctx, session, interaction, userHistoryServiceClient, tokenGenerator)
		case "list":
			handleListUserHistory(ctx, session, interaction, userHistoryServiceClient, tokenGenerator)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
