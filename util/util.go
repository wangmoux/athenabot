package util

import (
	"encoding/json"
	"fmt"
	"github.com/corona10/goimagehash"
	"github.com/rivo/uniseg"
	"github.com/sirupsen/logrus"
	"image/jpeg"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func GetFilePHash(url string) (uint64, error) {
	res, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Error(err)
		}
	}(res.Body)
	img, _ := jpeg.Decode(res.Body)
	PHash, _ := goimagehash.PerceptionHash(img)
	return PHash.GetHash(), nil
}

func LogMarshal(v any) string {
	s, _ := json.Marshal(v)
	return string(s)
}

func LogMarshalFn(v ...any) func() []any {
	return func() []any {
		res := make([]any, len(v))
		for i, s := range v {
			res[i] = LogMarshal(s)
		}
		return res
	}
}

func NumToStr[T int | float64 | int64 | uint64](num T) string {
	switch reflect.TypeOf(num).Kind() {
	case reflect.Int:
		return strconv.Itoa(int(num))
	case reflect.Int64:
		return strconv.FormatInt(int64(num), 10)
	case reflect.Float64:
		return strconv.FormatFloat(float64(num), 'f', 0, 64)
	case reflect.Uint64:
		return strconv.FormatUint(uint64(num), 10)
	}
	return ""
}

func StrBuilder(args ...string) string {
	var builder strings.Builder
	for _, i := range args {
		builder.WriteString(i)
	}
	return builder.String()
}

func TGNameWidth(name string) int {
	var width int
	gr := uniseg.NewGraphemes(name)
	for gr.Next() {
		if strings.Contains(fmt.Sprintf("%U", gr.Runes()), "U+1F") {
			width += len(gr.Runes()) * 2
			continue
		}
		width += 1

	}
	return width
}
