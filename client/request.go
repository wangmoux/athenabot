package client

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

type File struct {
	File     any
	FileName string
}

type RequestFD struct {
	Url     string
	Method  string
	Head    map[string]string
	Text    map[string]string
	File    *File
	TimeOut time.Duration
}

func NewRequestFD(url string, method string) *RequestFD {
	return &RequestFD{Url: url, Method: method}
}

func (r *RequestFD) set() {
	if r.TimeOut == 0 {
		r.TimeOut = time.Second * 3
	}
}

func (r *RequestFD) Do() ([]byte, error) {
	r.set()
	var res []byte
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range r.Text {
		_ = writer.WriteField(k, v)
	}
	if r.File != nil {
		switch reflect.TypeOf(r.File.File).Kind() {
		case reflect.String:
			_file, _ := r.File.File.(string)
			file, err := os.Open(_file)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			formDataFile, _ := writer.CreateFormFile(r.File.FileName, filepath.Base(_file))
			_, _ = io.Copy(formDataFile, file)
			_ = writer.Close()
		default:
			_file, _ := r.File.File.(io.Reader)

			formDataFile, _ := writer.CreateFormFile(r.File.FileName, r.File.FileName)
			_, _ = io.Copy(formDataFile, _file)
			_ = writer.Close()
		}
	}
	req, err := http.NewRequest(r.Method, r.Url, body)
	if err != nil {
		return res, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for k, v := range r.Head {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: r.TimeOut}
	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	res, _ = ioutil.ReadAll(resp.Body)
	return res, nil
}
