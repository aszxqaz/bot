package client

import (
	"encoding/json"
	"io"
	"log"
)

func readJson(r io.Reader, data any) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	log.Println("[JsonReader] Reading response: " + string(body))

	// decoder := json.NewDecoder(r)
	// err := decoder.Decode(data)

	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return err
}
