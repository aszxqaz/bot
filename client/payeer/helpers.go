package payeer

import (
	"encoding/json"
	"time"
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
