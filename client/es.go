package client

import (
	"athenabot/model"
	"context"
	"encoding/json"
	"github.com/olivere/elastic/v7"
)

type EsClient struct {
	*elastic.Client
	name string
}

func newEsClient(url string) (ImageDocClient, error) {
	es, err := elastic.NewClient(elastic.SetURL(url), elastic.SetSniff(false))
	if err != nil {
		return nil, err
	}
	return &EsClient{Client: es}, nil
}

func (e *EsClient) set() {
	e.name = model.ImageDocIndexName
}

func (e *EsClient) addImageDoc(imageDoc *model.ImageDoc) error {
	e.set()
	_, err := e.Index().Index(e.name).BodyJson(imageDoc).Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (e *EsClient) searchImageDoc(chatID int64, phrase string) ([]*model.ImageDoc, error) {
	e.set()
	var imageDocs []*model.ImageDoc
	defer func() {
		for _, i := range imageDocs {
			model.ImageDocPool.Put(i)
		}
	}()
	boolQuery := elastic.NewBoolQuery().Must(
		elastic.NewTermQuery("chat_id", chatID),
		elastic.NewMatchQuery("image_phrases", phrase))
	res, err := e.Search(e.name).Query(boolQuery).Size(searchLimit).Do(context.Background())
	if err != nil {
		return imageDocs, err
	}
	for _, item := range res.Hits.Hits {
		imageDoc := model.ImageDocPool.Get().(*model.ImageDoc)
		_ = json.Unmarshal(item.Source, imageDoc)
		imageDocs = append(imageDocs, imageDoc)

	}
	return imageDocs, nil
}
