package effects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// SourceRevision returns a stable digest for an authored Block v2 definition.
// Products store it with composition provenance when they pin an active snapshot.
func SourceRevision(blocks []Block) (string, error) {
	canonical, err := json.Marshal(BlockSet(blocks))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}
