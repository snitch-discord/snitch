package slashcommand

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"snitch/internal/shared/ctxutil"

	"github.com/bwmarrin/discordgo"
)

func WithTimeout(next SlashCommandHandlerFunc, duration time.Duration) SlashCommandHandlerFunc {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		timeoutCtx, cancel := context.WithTimeout(ctx, duration)
		defer cancel()
		next(timeoutCtx, session, interaction)
	}
}

func Log(next SlashCommandHandlerFunc) SlashCommandHandlerFunc {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With(
			slog.String("User ID", interaction.Member.User.ID),
			slog.String("Guild ID", interaction.GuildID),
			slog.String("Command", interaction.ApplicationCommandData().Name),
		)

		ctx = ctxutil.WithValue(ctx, slogger)
		next(ctx, session, interaction)
	}
}

func ResponseTime(next SlashCommandHandlerFunc) SlashCommandHandlerFunc {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		start := time.Now()
		next(ctx, session, interaction)
		elapsed := time.Since(start)

		slogger.InfoContext(ctx, "Command Time", "Time Elapsed", elapsed)
	}
}

func Recovery(next SlashCommandHandlerFunc) SlashCommandHandlerFunc {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		defer func() {
			if err := recover(); err != nil {
				slogger, ok := ctxutil.Value[*slog.Logger](ctx)
				if !ok {
					slogger = slog.Default()
				}
				stack := debug.Stack()

				slogger.ErrorContext(ctx, "Recovery", "Panic", err, "Stack", stack)
			}
		}()
		next(ctx, session, interaction)
	}
}
