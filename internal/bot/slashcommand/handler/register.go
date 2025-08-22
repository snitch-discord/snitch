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

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func handleCreateGroup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.RegistrarServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options[0].Options

	userID := interaction.Member.User.ID

	groupName := options[0].StringValue()

	registerRequest := connect.NewRequest(&snitchv1.RegisterRequest{UserId: userID, GroupName: &groupName})
	registerResponse, err := client.Register(ctx, registerRequest)

	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't register group, error: %s", err.Error()))
		return
	}

	messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Created group %s for this server.", registerResponse.Msg.GroupId))
}

func handleJoinGroup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.RegistrarServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options[0].Options

	userID := interaction.Member.User.ID
	groupId := options[0].StringValue()
	err := uuid.Validate(groupId)

	if err != nil {
		slogger.ErrorContext(ctx, "invalid group id", "error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Invalid group ID: %s", groupId))
		return
	}

	registerRequest := connect.NewRequest(&snitchv1.RegisterRequest{UserId: userID, GroupId: &groupId})
	registerResponse, err := client.Register(ctx, registerRequest)

	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Couldn't join group, error: %s", err.Error()))
		return
	}

	messageutil.SimpleRespondContext(ctx, session, interaction, fmt.Sprintf("Joined group %s", registerResponse.Msg.GroupId))
}

func handleGroupCommands(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, client snitchv1connect.RegistrarServiceClient) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options

	switch options[0].Name {
	case "create":
		handleCreateGroup(ctx, session, interaction, client)
	case "join":
		handleJoinGroup(ctx, session, interaction, client)
	default:
		slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[1].Name)
	}
}

func CreateRegisterCommandHandler(botconfig botconfig.BotConfig, httpClient http.Client) slashcommand.SlashCommandHandlerFunc {
	backendURL, err := botconfig.BackendURL()
	if err != nil {
		log.Fatal(backendURL)
	}

	registrarServiceClient := snitchv1connect.NewRegistrarServiceClient(&httpClient, backendURL.String())

	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		getGroupReq := connect.NewRequest(&snitchv1.GetGroupForServerRequest{ServerId: interaction.GuildID})
		// We expect this to fail when creating or joining a group for the first time.
		getGroupResp, err := registrarServiceClient.GetGroupForServer(ctx, getGroupReq)
		groupID := ""
		if err == nil {
			groupID = getGroupResp.Msg.GroupId
		}

		ctx = transport.WithAuthInfo(ctx, interaction.GuildID, groupID)

		options := interaction.ApplicationCommandData().Options

		switch options[0].Name {
		case "group":
			handleGroupCommands(ctx, session, interaction, registrarServiceClient)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
