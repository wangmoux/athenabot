package model

import (
	"sync"
	"time"
)

const (
	ImageDocIndexName = "mars"
)

type ImageDoc struct {
	ImagePhrases []string  `json:"image_phrases" bson:"image_phrases"`
	MarsID       string    `json:"mars_id" bson:"mars_id"`
	ChatID       int64     `json:"chat_id" bson:"chat_id"`
	CreateTime   time.Time `json:"create_time" bson:"create_time"`
}

type MysqlImageDoc struct {
	ImagePhrases string
	MarsID       string
	ChatID       int64
	CreateTime   string
}

func (MysqlImageDoc) TableName() string {
	return ImageDocIndexName
}

var ImageDocPool = &sync.Pool{
	New: func() any {
		return new(ImageDoc)
	},
}
