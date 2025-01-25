package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/shared/ctxutil"

	"github.com/bwmarrin/discordgo"
)

type RegistrationRequest struct {
	ServerID  string `json:"serverId"` // we need to tell go that our number is encoded as a string, hence ',string'
	UserID    string `json:"userId"`   // we need to tell go that our number is encoded as a string, hence ',string'
	GroupName string `json:"groupName,omitempty"`
}

type RegistrationResponse struct {
	ServerID string `json:"serverId"` // we need to tell go that our number is encoded as a string, hence ',string'
	GroupID  string `json:"groupId"`
}

func handleCreateGroup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, backendURL *url.URL) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options[0].Options

	serverID := interaction.GuildID
	userID := interaction.Member.User.ID
	groupName := options[0].StringValue()

	requestStruct := &RegistrationRequest{ServerID: serverID, UserID: userID, GroupName: groupName}

	requestBody, err := json.Marshal(requestStruct)
	if err != nil {
		log.Print(err)
		return
	}

	requestURL := backendURL.JoinPath("databases")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Creation", "Error", err)
		return
	}

	request.Header.Add("X-Server-ID", interaction.GuildID)

	response, err := session.Client.Do(request)
	if err != nil {
		slogger.ErrorContext(ctx, "Backend Request Call", "Error", err)
		return
	}

	if response.StatusCode >= 300 || response.StatusCode < 200 {
		body, _ := io.ReadAll(response.Body)
		defer response.Body.Close()
		slogger.ErrorContext(ctx, "Unexpected Response", "Status", response.StatusCode, "Body", string(body))
	
		if err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Couldn't register group, error: %s", body),
			},
		}); err != nil {
			slogger.ErrorContext(ctx, "Couldn't Write Discord Response", "Error", err)
		}

		return
	}

	body, err := io.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		slogger.ErrorContext(ctx, "Couldn't Read Body", "Error", err)
		return
	}

	var registrationResponse RegistrationResponse
	if err := json.Unmarshal(body, &registrationResponse); err != nil {
		slogger.ErrorContext(ctx, "Couldn't Unmarshal Body", "Error", err)
		return
	}

	if err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Created group %s for this server.", registrationResponse.GroupID),
		},
	}); err != nil {
		slogger.ErrorContext(ctx, "Couldn't Write Discord Response", "Error", err)
		return
	}
}

func handleJoinGroup(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, backendURL *url.URL) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options

	slogger.DebugContext(ctx, "Join Options", "Options", options, "Session", session, "URL", backendURL)

	// TODO: implement
}

func handleGroupCommands(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, backendURL *url.URL) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	options := interaction.ApplicationCommandData().Options[0].Options

	switch options[0].Name {
	case "create":
		handleCreateGroup(ctx, session, interaction, backendURL)
	case "join":
		handleJoinGroup(ctx, session, interaction, backendURL)
	default:
		slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[1].Name)
	}

}

func CreateRegisterCommandHandler(botconfig botconfig.BotConfig) slashcommand.SlashCommandHandlerFunc {
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
		case "group":
			handleGroupCommands(ctx, session, interaction, backendURL)
		default:
			slogger.ErrorContext(ctx, "Invalid subcommand", "Subcommand Name", options[0].Name)
		}
	}
}
