package internal

import (
	"crypto/sha1"
	"encoding/hex"
)

func ComputeWebhookID(secretToken, salt string) string {
	h := sha1.New()
	h.Write([]byte(secretToken + salt))
	return hex.EncodeToString(h.Sum(nil))
}
