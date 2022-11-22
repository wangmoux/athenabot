package app

import (
	"athenabot/client"
	"athenabot/config"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
)

func (c Webhook) setCustomWebhook() (string, error) {
	req := client.NewRequestFD("https://api.telegram.org/bot"+config.Conf.BotToken+"/setWebhook", "POST")
	formData := make(map[string]string)
	formData["url"] = config.Conf.Webhook.Endpoint + config.Conf.Webhook.Token
	req.FormData = formData
	req.File = &client.File{
		File:     config.Conf.Webhook.CertFile,
		FileName: "certificate",
	}
	res, err := req.Do()
	if err != nil {
		return "", err
	}
	return string(res), nil
}

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
