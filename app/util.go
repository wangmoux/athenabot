package app

import (
	"athenabot/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (c Webhook) setWebhook() (string, error) {
	certFile, err := ioutil.ReadFile(config.Conf.Webhook.CertFile)
	if err != nil {
		return "", err
	}
	cert := tgbotapi.FileBytes{
		Name:  "certificate",
		Bytes: certFile,
	}
	wh, _ := tgbotapi.NewWebhookWithCert(config.Conf.Webhook.Endpoint+config.Conf.Webhook.Token, cert)
	_, err = c.bot.Request(wh)
	if err != nil {
		return "", err
	}
	info, err := c.bot.GetWebhookInfo()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%+v", info), err
}

type DebugMessageRequest struct {
	Message     *tgbotapi.Update `json:"message,omitempty"`
	MessageType string           `json:"message_type"`
}

func debugMessage(u *tgbotapi.Update) {
	ds := os.Getenv("DEBUG_SERVER")
	if ds == "" {
		return
	}
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	payload, _ := json.Marshal(DebugMessageRequest{
		Message:     u,
		MessageType: "tg_update",
	})
	req, err := http.NewRequest("POST", ds, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		logrus.Errorf("debug_message error: %s", err.Error())
		return
	}
}
