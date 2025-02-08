package messageutil

import (
	"context"
	"log/slog"
	"snitch/internal/shared/ctxutil"

	"github.com/bwmarrin/discordgo"
)

func SimpleRespondContext(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, messageContent string) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()

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
