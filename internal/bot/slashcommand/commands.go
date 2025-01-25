package slashcommand

import "github.com/bwmarrin/discordgo"

func InitializeCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "register",
			Description: "Registers server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "group",
					Description: "Group related functionality",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "create",
							Description: "Creates a new group",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "group-name",
									Type:        discordgo.ApplicationCommandOptionString,
									Description: "Group name",
									Required:    true,
								},
							},
						},
						{
							Name:        "join",
							Description: "Joins a group",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "join-code",
									Type:        discordgo.ApplicationCommandOptionString,
									Description: "Group join code",
									Required:    true,
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "report",
			Description: "Reports a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "new",
					Description: "Creates a new report for the targeted user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "reported-user",
							Type:        discordgo.ApplicationCommandOptionUser,
							Description: "The user to report",
							Required:    true,
						},
						{
							Name:        "report-reason",
							Type:        discordgo.ApplicationCommandOptionString,
							Description: "Report reason",
							Required:    false,
						},
					},
				},
				{
					Name:        "list",
					Description: "Retrieves reports",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "reported-user",
							Type:        discordgo.ApplicationCommandOptionUser,
							Description: "The user to retrieve reports for",
							Required:    false,
						},
						{
							Name:        "reporter-user",
							Type:        discordgo.ApplicationCommandOptionUser,
							Description: "The user who created the reports",
							Required:    false,
						},
					},
				},
				{
					Name:        "delete",
					Description: "Deletes a report",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "report-id",
							Type:        discordgo.ApplicationCommandOptionString,
							Description: "Report ID",
							Required:    true,
						},
					},
				},
			},
		},
	}
}
