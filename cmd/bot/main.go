package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"snitch/internal/bot/auth"
	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/events"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/bot/slashcommand/handler"
	"snitch/internal/bot/slashcommand/middleware"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"github.com/bwmarrin/discordgo"
)

func main() {
	config, err := botconfig.FromEnv()
	if err != nil {
		log.Fatalf("Failed to load bot configuration from environment: %v", err)
	}

	// Load CA certificate for backend service validation
	caCert, err := os.ReadFile(config.CaCertPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatalf("Failed to parse CA certificate")
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	tokenGenerator := auth.NewTokenGenerator(config.JwtSecret)

	// initialize map of command name to command handler
	commandHandlers := map[string]slashcommand.SlashCommandHandlerFunc{
		"register": handler.CreateRegisterCommandHandler(config, httpClient, tokenGenerator),
		"report":   handler.CreateReportCommandHandler(config, httpClient, tokenGenerator),
		"user":     handler.CreateUserCommandHandler(config, httpClient, tokenGenerator),
	}

	commands := slashcommand.InitializeCommands()

	for _, command := range commands {
		_, handlerPresent := commandHandlers[command.Name]

		if !handlerPresent {
			log.Fatalf("Missing Handler for %s", command.Name)
		}
	}

	mainSession, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}
	defer func() {
		if err := mainSession.Close(); err != nil {
			log.Printf("Failed to close Discord session: %v", err)
		}
	}()

	slogger := slog.Default()
	backendURL, err := config.BackendURL()
	if err != nil {
		log.Fatalf("Failed to get backend URL: %v", err)
	}

	eventClient := events.NewClient(backendURL.String(), config.JwtSecret, mainSession, slogger, &httpClient)

	eventClient.RegisterHandler(snitchv1.EventType_EVENT_TYPE_REPORT_CREATED, events.CreateReportCreatedHandler(slogger))
	eventClient.RegisterHandler(snitchv1.EventType_EVENT_TYPE_REPORT_DELETED, events.CreateReportDeletedHandler(slogger))
	eventClient.RegisterHandler(snitchv1.EventType_EVENT_TYPE_USER_BANNED, events.CreateUserBannedHandler(slogger))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventClient.Start(ctx)
	defer eventClient.Stop()

	mainSession.AddHandler(func(session *discordgo.Session, ready *discordgo.Ready) {
		slogger.InfoContext(ctx, "logged in", "username", fmt.Sprintf("%s#%s", session.State.User.Username, session.State.User.Discriminator))

		// Initialize event subscriptions for all guilds the bot is already in
		slogger.Info("Initializing event subscriptions for existing guilds", "guild_count", len(ready.Guilds))
		for _, guild := range ready.Guilds {
			if err := eventClient.AddServer(ctx, guild.ID); err != nil {
				slogger.Error("Failed to initialize server subscription", "guild_id", guild.ID, "error", err)
			} else {
				slogger.Debug("Initialized server subscription", "guild_id", guild.ID)
			}
		}
	})
	// setup our listeners for interaction events (a user using a slash command)

	handler := func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			handler(ctx, session, interaction)
		}
	}
	handler = middleware.RequireManageServer(handler)
	handler = middleware.ResponseTime(handler)
	handler = middleware.Recovery(handler)
	handler = middleware.Log(handler)
	handler = middleware.WithTimeout(handler, time.Second*10)
	mainSession.AddHandler(slashcommand.SlashCommandHandlerFunc(handler).Adapt())

	if err = mainSession.Open(); err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}

	// Add event handlers for when bot joins/leaves servers
	mainSession.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		slogger.Info("Bot joined server", "guild_id", g.ID, "guild_name", g.Name)
		if err := eventClient.AddServer(ctx, g.ID); err != nil {
			slogger.Error("Failed to add server to event subscription", "guild_id", g.ID, "error", err)
		}
	})

	mainSession.AddHandler(func(s *discordgo.Session, g *discordgo.GuildDelete) {
		slogger.Info("Bot left server", "guild_id", g.ID)
		eventClient.RemoveServer(g.ID)
	})

	// Register commands globally (works across all servers)
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	slogger.Info("Registering global slash commands", "command_count", len(commands))
	for index, applicationCommand := range commands {
		createdCommand, err := mainSession.ApplicationCommandCreate(mainSession.State.User.ID, "", applicationCommand)
		if err != nil {
			log.Fatalf("Cannot register '%v' command: %v", applicationCommand.Name, err)
		}

		registeredCommands[index] = createdCommand
		slogger.Info("Registered command", "command_name", applicationCommand.Name)
	}

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, os.Interrupt)
	<-stopChannel

	slogger.Info("Shutting down gracefully...")

	// cleanup global commands
	slogger.Info("Cleaning up global slash commands")
	for _, registeredCommand := range registeredCommands {
		if err = mainSession.ApplicationCommandDelete(mainSession.State.User.ID, "", registeredCommand.ID); err != nil {
			log.Printf("Cannot delete '%v' command: '%v'", registeredCommand.Name, err)
		}
	}
}
