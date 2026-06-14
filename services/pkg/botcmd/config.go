// Package botcmd configures Matrix room bot command prefixes (BOT_COMMAND_PREFIX).
package botcmd

// Config maps BOT_COMMAND_PREFIX to bot.command.prefix.
type Config struct {
	Command struct {
		Prefix string `default:"!edel"`
	}
}
