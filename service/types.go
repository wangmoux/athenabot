package service

import "sync"

const (
	marsKeyDir                     = "bot:mars_data:"
	marsTopKeyDir                  = "bot:mars_top:"
	studyTopKeyDir                 = "bot:study_top:"
	warnKeyDir                     = "bot:warn_data:"
	chatVerifyKeyDir               = "bot:chat_verify:"
	serviceSwitchKeyDir            = "bot:service_switch:"
	commandLimitKeyDir             = "bot:command_limit:"
	deleteMessageKeyDir            = "bot:delete_message:"
	doudouTopKeyDir                = "bot:doudou_top:"
	chat48hMessageDir              = "bot:chat_48h_message:"
	chat48hMessageDeleteCrontabDir = "bot:chat_48h_message_delete_crontab:"
)

type groupAdministratorsCache map[int64]uint8

var (
	commandsFunc              = make(map[string]func(c *CommandConfig))
	groupsAdministratorsCache = make(map[int64]groupAdministratorsCache)
	userNameCache             = make(map[int64]string)
	unknownUserCache          = make(map[int64]uint8)
	groupsChatLimit           = make(map[int64]*chatLimit)
	userNameCacheLock         sync.RWMutex
)
