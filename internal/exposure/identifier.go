package exposure

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func stableID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:12])
}
