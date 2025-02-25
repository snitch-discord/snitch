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
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
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
	registerRequest.Header().Add("X-Server-ID", interaction.GuildID)
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

	options := interaction.ApplicationCommandData().Options

	slogger.DebugContext(ctx, "Join Options", "Options", options, "Session", session, "Client", client)

	// TODO: implement
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

		options := interaction.ApplicationCommandData().Options

		switch options[0].Name {
		case "group":
			handleGroupCommands(ctx, session, interaction, registrarServiceClient)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
