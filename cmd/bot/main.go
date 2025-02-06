package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"snitch/internal/bot/botconfig"
	"snitch/internal/bot/slashcommand"
	"snitch/internal/bot/slashcommand/handler"
	"snitch/internal/bot/slashcommand/middleware"

	"github.com/bwmarrin/discordgo"
)

func main() {
	testingGuildID := "1315524176936964117"

	config, err := botconfig.FromEnv()
	if err != nil {
		log.Panic(err)
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	// initialize map of command name to command handler
	commandHandlers := map[string]slashcommand.SlashCommandHandlerFunc{
		"register": handler.CreateRegisterCommandHandler(config, httpClient),
		"report":   handler.CreateReportCommandHandler(config, httpClient),
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
		log.Panic(err)
	}
	defer mainSession.Close()

	mainSession.AddHandler(func(session *discordgo.Session, _ *discordgo.Ready) {
		log.Printf("Logged in as: %s#%s", session.State.User.Username, session.State.User.Discriminator)
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
		log.Panic(err)
	}

	// tells discord about the commands we support
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	for index, applicationCommand := range commands {
		createdCommand, err := mainSession.ApplicationCommandCreate(mainSession.State.User.ID, testingGuildID, applicationCommand)
		if err != nil {
			log.Panicf("Cannot register '%v' command: %v", applicationCommand.Name, err)
		}

		registeredCommands[index] = createdCommand
	}

	if err != nil {
		log.Panic(err)
	}

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, os.Interrupt)
	<-stopChannel

	log.Println("Shutting down gracefully...")

	// cleanup commands
	for _, registeredCommand := range registeredCommands {
		if err = mainSession.ApplicationCommandDelete(mainSession.State.User.ID, testingGuildID, registeredCommand.ID); err != nil {
			log.Panicf("Cannot delete '%v' command: '%v'", registeredCommand.Name, err)
		}
	}
}
