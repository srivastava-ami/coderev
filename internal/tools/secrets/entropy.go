package secrets

import (
	"math"
	"regexp"
)

// entropyThreshold is the minimum Shannon entropy (bits per character) a
// candidate string must have before the generic, keyword-gated detector treats
// it as a likely secret. 3.5 comfortably clears random base64/hex-mixed tokens
// (~4.5+) while staying above repetitive natural-language strings (~2.7).
const entropyThreshold = 3.5

// minGenericSecretLen is the shortest value the generic detector considers.
// UUIDs (36) and common hex digests (32/40/64) are longer but excluded by
// shape, so this only guards against short, low-signal strings.
const minGenericSecretLen = 20

// reUUID matches a canonical RFC-4122 UUID — a frequent high-entropy false
// positive (ids, fixtures) that is never a credential.
var reUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// reHexDigest matches an all-hex string. md5/sha1/sha256 digests and hex ids
// land here; they show up constantly in test fixtures and are excluded so the
// generic detector does not flag "test hashes".
var reHexDigest = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// shannonEntropy returns the Shannon entropy of s in bits per character.
func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	var counts [256]float64
	for i := 0; i < len(s); i++ {
		counts[s[i]]++
	}
	n := float64(len(s))
	var h float64
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := c / n
		h -= p * math.Log2(p)
	}
	return h
}

// looksLikeSecret decides whether a candidate value (already stripped of its
// surrounding quotes) is a likely high-entropy credential. It deliberately
// excludes UUIDs and hex digests so test hashes and ids are never flagged.
func looksLikeSecret(s string) bool {
	if len(s) < minGenericSecretLen {
		return false
	}
	if reUUID.MatchString(s) {
		return false
	}
	if reHexDigest.MatchString(s) {
		return false
	}
	return shannonEntropy(s) >= entropyThreshold
}
