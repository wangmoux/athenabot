package service

import (
	"athenabot/config"
	"strconv"
	"strings"
)

func (c *CommandConfig) startCommand() {
	if c.commandArg == "" {
		if len(c.update.Message.Text) < 1 {
			c.messageConfig.Text = "Hello"
		}
		if config.Conf.Modules.EnablePrivateChat {
			NewChatPrivateConfig(c.BotConfig).ChatBotHandler()
			return
		}
	}
	commandArgs := strings.FieldsFunc(c.commandArg, func(r rune) bool {
		return r == '_'
	})
	if len(commandArgs) == 2 {
		switch commandArgs[0] {
		case "verify":
			c.commandArg = commandArgs[1]
			chatID, err := strconv.Atoi(c.commandArg)
			if err != nil {
				return
			}
			NewChatConfig(c.BotConfig).chatMemberVerify(int64(chatID))
		default:
			return
		}
	}
}
