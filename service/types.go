package service

const (
	marsKeyDir          = "bot:mars_data:"
	marsTopKeyDir       = "bot:mars_top:"
	studyTopKeyDir      = "bot:study_top:"
	warnKeyDir          = "bot:warn_data:"
	chatVerifyKeyDir    = "bot:chat_verify:"
	commandSwitchKeyDir = "bot:command_switch:"
	commandLimitKeyDir  = "bot:command_limit:"
)

var commandsFunc = make(map[string]func(c *CommandConfig))

type groupAdministratorsCache map[int64]uint8

var groupsAdministratorsCache = make(map[int64]groupAdministratorsCache)

var userNameCache = make(map[int64]string)

var unknownUserCache = make(map[int64]uint8)

type chatLimit struct {
	userID    int64
	count     int
	timestamp int64
}

var groupsChatLimit = make(map[int64]*chatLimit)