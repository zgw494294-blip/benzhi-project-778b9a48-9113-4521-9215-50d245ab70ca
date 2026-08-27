package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Digest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
