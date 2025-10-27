package service

import (
	"athenabot/config"
	"athenabot/model"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ChatPrivateConfig struct {
	*BotConfig
}

func NewChatPrivateConfig(botConfig *BotConfig) *ChatPrivateConfig {
	return &ChatPrivateConfig{BotConfig: botConfig}
}

var chatBotPrivateSessions sync.Map

func (c *ChatPrivateConfig) ChatBotHandler() {
	if len(c.update.Message.Text) < 1 || len(c.update.Message.Text) > 1024 {
		return
	}
	_contents, ok := chatBotPrivateSessions.Load(c.chatID)
	if !ok {
		_contents = []*model.ChatbotContent{}
	}
	contents := _contents.([]*model.ChatbotContent)
	defer func() {
		chatBotPrivateSessions.Store(c.chatID, contents)
	}()
	contents = append(contents, &model.ChatbotContent{
		Nickname: c.update.Message.From.FirstName,
		Content:  c.update.Message.Text,
	})
	chatBotPrivateSessionsLen := len(contents)
	if chatBotPrivateSessionsLen > 10 {
		contents = contents[chatBotPrivateSessionsLen-10:]
	}
	logrus.Infof("chat_bot_private request for %+v", c.update.Message.Text)
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	payload, _ := json.Marshal(model.ChatbotRequest{
		ChatType: "chat_bot",
		Contents: contents,
	})
	req, err := http.NewRequest("POST", config.Conf.ChatBot.ChatBotServerURL, bytes.NewReader(payload))
	if err != nil {
		logrus.Error(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return
	}
	if resp.StatusCode != 200 {
		logrus.Error(resp.StatusCode)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("chat_bot_private: %s by %s", body, c.update.Message.Text)
	chatbotMessage := model.ChatbotResponse{}
	_ = json.Unmarshal(body, &chatbotMessage)
	c.messageConfig.Text = chatbotMessage.BotMessage
	c.sendMessage()
	contents = append(contents, &model.ChatbotContent{
		IsModel:  true,
		Nickname: c.bot.Self.FirstName,
		Content:  chatbotMessage.BotMessage,
	})
}
