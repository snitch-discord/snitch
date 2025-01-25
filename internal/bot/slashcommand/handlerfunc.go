package slashcommand

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type SlashCommandHandlerFunc func(context.Context, *discordgo.Session, *discordgo.InteractionCreate)

func (slashCommandFuncContext SlashCommandHandlerFunc) Adapt() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slashCommandFuncContext(context.Background(), session, interaction)
	}
}
