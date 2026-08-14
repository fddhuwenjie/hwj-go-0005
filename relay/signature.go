package relay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func Verify(secret string, payload []byte, provided string) bool {
	expected := Sign(secret, payload)
	return hmac.Equal([]byte(expected), []byte(provided))
}
