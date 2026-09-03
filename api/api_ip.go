package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func HashIPAddress(ip string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}
