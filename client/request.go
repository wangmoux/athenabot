package client

import (
	"athenabot/config"
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Request struct {
	Url    string
	Method string
	Body   string
	Heads  map[string]string
}

func NewRequest(url string, method string) *Request {
	return &Request{Url: url, Method: method}
}

func (h Request) Do() ([]byte, error) {
	var res []byte
	req, err := http.NewRequest(h.Method, h.Url, strings.NewReader(h.Body))
	if err != nil {
		return res, err
	}
	for k, v := range h.Heads {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: time.Second * 3}
	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	res, _ = ioutil.ReadAll(resp.Body)
	return res, nil
}

type File struct {
	File     string
	FileName string
}

type RequestFD struct {
	Url      string
	Method   string
	Heads    map[string]string
	FormData map[string]string
	File     *File
}

func NewRequestFD(url string, method string) *RequestFD {
	return &RequestFD{Url: url, Method: method}
}

func (h RequestFD) Do() ([]byte, error) {
	var res []byte
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range h.FormData {
		_ = writer.WriteField(k, v)
	}
	if h.File != nil {
		file, err := os.Open(h.File.File)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		formDataFile, _ := writer.CreateFormFile(h.File.FileName, filepath.Base(config.Conf.Webhook.CertFile))
		_, _ = io.Copy(formDataFile, file)
		_ = writer.Close()
	}
	req, err := http.NewRequest(h.Method, h.Url, body)
	if err != nil {
		return res, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for k, v := range h.Heads {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: time.Second * 3}
	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	res, _ = ioutil.ReadAll(resp.Body)
	return res, nil
}
