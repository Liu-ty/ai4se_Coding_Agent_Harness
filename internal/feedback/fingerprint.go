package feedback

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func Fingerprint(stageID, category, text string) string {
	normalized := normalizeFingerprintText(text)
	sum := sha256.Sum256([]byte(strings.Join([]string{stageID, category, normalized}, "\x00")))
	return hex.EncodeToString(sum[:])
}
