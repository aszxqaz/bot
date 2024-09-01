package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(secret []byte, data []byte) string {
	hmac := hmac.New(sha256.New, secret)
	hmac.Write([]byte(data))
	dataHmac := hmac.Sum(nil)

	hmacHex := hex.EncodeToString(dataHmac)

	return hmacHex
}
