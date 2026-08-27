package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

var clock = func() time.Time { return time.Now().UTC() }

func newID(prefix string) string {
	var b [12]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
