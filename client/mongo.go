package client

import (
	"athenabot/model"
	"athenabot/util"
	"context"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	*mongo.Client
	name string
}

func newMongodbClient(url string) (ImageDocClient, error) {
	opt := options.Client().ApplyURI(url).
		SetMinPoolSize(5).SetMaxPoolSize(100)
	db, err := mongo.Connect(context.Background(), opt)
	if err != nil {
		return nil, err
	}
	return &MongoClient{Client: db}, nil
}

func (m *MongoClient) set() {
	m.name = model.ImageDocIndexName
}

func (m *MongoClient) addImageDoc(doc *model.ImageDoc) error {
	m.set()
	coll := m.Client.Database(m.name).Collection(m.name)
	_, err := coll.InsertOne(context.Background(), doc)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoClient) searchImageDoc(chatID int64, phrase string) ([]*model.ImageDoc, error) {
	m.set()
	coll := m.Client.Database(m.name).Collection(m.name)
	var imageDocs []*model.ImageDoc
	filter := bson.D{{"chat_id", chatID},
		{"$text", bson.D{{"$search", util.StrBuilder("\"", phrase, "\"")}}}}
	cursor, err := coll.Find(context.Background(), filter)
	if err != nil {
		return imageDocs, err
	}
	for cursor.Next(context.Background()) {
		imageDoc := model.ImageDocPool.Get().(*model.ImageDoc)
		model.ImageDocPool.Put(imageDoc)
		err = cursor.Decode(imageDoc)
		if err != nil {
			logrus.Error(err)
			continue
		}
		imageDocs = append(imageDocs, imageDoc)
	}
	return imageDocs, err
}
