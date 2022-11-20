package app

import (
	"athenabot/config"
	"bytes"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func setWebhook() {
	url := "https://api.telegram.org/bot" + config.Conf.BotToken + "/setWebhook"
	method := "POST"
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	certFile, err := os.Open(config.Conf.Webhook.CertFile)
	if err != nil {
		logrus.Error(err)
	}
	defer certFile.Close()

	certificate, _ := writer.CreateFormFile("certificate", filepath.Base(config.Conf.Webhook.CertFile))
	_, _ = io.Copy(certificate, certFile)
	_ = writer.WriteField("url", config.Conf.Webhook.Endpoint+config.Conf.Webhook.Token)
	_ = writer.Close()

	client := &http.Client{}
	req, _ := http.NewRequest(method, url, payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
	}

	defer resp.Body.Close()
	res, _ := ioutil.ReadAll(resp.Body)

	logrus.Infof("set_webhook_res:%v", string(res))
}
