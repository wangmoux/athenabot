package service

import (
	"strconv"
	"strings"
)

func (c *CommandConfig) startCommand() {
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
			NewChatMemberConfig(c.ctx, c.BotConfig).chatMemberVerify(int64(chatID))
		}
	}
}
