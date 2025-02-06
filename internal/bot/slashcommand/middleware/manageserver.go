package middleware

import (
	"context"
	"log/slog"
	"strconv"

	"snitch/internal/bot/slashcommand"
	"snitch/internal/shared/ctxutil"

	"github.com/bwmarrin/discordgo"
)

const MANAGE_SERVER_BIT = 5

func RequireManageServer(next slashcommand.SlashCommandHandlerFunc) slashcommand.SlashCommandHandlerFunc {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		slogger.InfoContext(ctx, "Permissions", "Member", strconv.FormatInt(interaction.Member.Permissions, 2), "Manage Server", strconv.FormatInt(MANAGE_SERVER_BIT, 2))

		if MANAGE_SERVER_BIT == (interaction.Member.Permissions & MANAGE_SERVER_BIT) {
			next(ctx, session, interaction)
		} else {
			if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You are not allowed to use this command!",
				},
			}); err != nil {
				slogger.ErrorContext(ctx, "Couldn't Write Discord Response", "Error", err)
			}
		}
	}
}
