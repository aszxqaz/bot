package payeer

import (
	"encoding/json"
	"strings"
	"time"

	fastshot "github.com/opus-domini/fast-shot"
	"github.com/opus-domini/fast-shot/constant/mime"
)

func mustMarshalJson(data any) []byte {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return body
}

func getTimestamp() int64 {
	return time.Now().UnixMilli()
}

func joinPairs(pairs []Pair) string {
	joined := strings.Builder{}
	for i, pair := range pairs {
		joined.WriteString(string(pair))
		if i != len(pairs)-1 {
			joined.WriteString(",")
		}
	}
	return joined.String()
}

func setupHttpClient(apiId string) fastshot.ClientHttpMethods {
	return fastshot.NewClient(baseUrl).
		Header().Add("API-ID", apiId).
		Header().AddAccept(mime.JSON).
		Build()
}
