package client

import (
	"athenabot/model"
	"fmt"
	"testing"
	"time"
)

func TestAddImageDoc(t *testing.T) {
	//c := ImageDocProvider["es"]("http://localhost:9200")
	c := ImageDocProvider["mongo"]("mongodb://localhost:27017")
	//c := ImageDocProvider["mysql"]("root:password@tcp(localhost:3306)/mars")
	err := c.addImageDoc(&model.ImageDoc{
		ImagePhrases: []string{"哈哈哈", "你好我好"},
		MarsID:       "13712198688194467123",
		ChatID:       -1001546229241,
		CreateTime:   time.Now().UTC(),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSearchImageDoc(t *testing.T) {
	//c := ImageDocProvider["es"]("http://localhost:9200")
	c := ImageDocProvider["mongo"]("mongodb://localhost:27017")
	//c := ImageDocProvider["mysql"]("root:password@tcp(localhost:3306)/mars")
	res, err := c.searchImageDoc(-1001546229241, "哈哈哈")
	if err != nil {
		t.Error(err)
	}
	for _, i := range res {
		fmt.Println(i)
	}
}

func BenchmarkAddImageDoc(b *testing.B) {
	//c := ImageDocProvider["es"]("http://localhost:9200")
	c := ImageDocProvider["mongo"]("mongodb://localhost:27017")
	//c := ImageDocProvider["mysql"]("root:password@tcp(localhost:3306)/mars")
	for i := 0; i < b.N; i++ {
		err := c.addImageDoc(&model.ImageDoc{
			ImagePhrases: []string{"哈哈哈", "你好我好"},
			MarsID:       "13712198688194467123",
			ChatID:       -1001546229241,
			CreateTime:   time.Now().UTC(),
		})
		if err != nil {
			b.Error(err)
		}
	}
}
