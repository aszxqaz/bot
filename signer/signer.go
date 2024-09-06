package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(bytes []byte, secret []byte) string {
	hmac := hmac.New(sha256.New, secret)
	hmac.Write(bytes)
	dataHmac := hmac.Sum(nil)
	hmacHex := hex.EncodeToString(dataHmac)
	return hmacHex
}
