// Package dht provides DHT (Distributed Hash Table) functionality for the LibreSeed seeder.
// It implements the DHT key generation, data structures, and operations as specified
// in the LibreSeed protocol specification v1.3.
package dht

import (
	"crypto/sha256"
	"encoding/base64"
)

// DHTKeySize is the size of DHT keys in bytes (SHA-256 truncated to 20 bytes for BitTorrent DHT).
const DHTKeySize = 20

// Key represents a 20-byte DHT key (info hash format for BitTorrent DHT).
type Key [DHTKeySize]byte

// KeyPrefix constants define the prefixes used for different DHT key types.
const (
	// PrefixManifest is the prefix for version-specific manifest keys.
	// Format: "libreseed:manifest:<name>@<version>"
	PrefixManifest = "libreseed:manifest:"

	// PrefixNameIndex is the prefix for package name index keys.
	// Format: "libreseed:name-index:<name>"
	PrefixNameIndex = "libreseed:name-index:"

	// PrefixAnnounce is the prefix for publisher announce keys.
	// Format: "libreseed:announce:<base64(pubkey)>"
	PrefixAnnounce = "libreseed:announce:"

	// PrefixSeeder is the prefix for seeder status keys.
	// Format: "libreseed:seeder:<seederID>"
	PrefixSeeder = "libreseed:seeder:"
)

// ManifestKey generates a DHT key for a specific package version manifest.
//
// The key is computed as:
//
//	sha256("libreseed:manifest:" + name + "@" + version) truncated to 20 bytes
//
// Example:
//
//	key := ManifestKey("mypackage", "1.4.0")
func ManifestKey(name, version string) Key {
	input := PrefixManifest + name + "@" + version
	return hashToKey(input)
}

// NameIndexKey generates a DHT key for a package's name index.
//
// The key is computed as:
//
//	sha256("libreseed:name-index:" + name) truncated to 20 bytes
//
// The name index enables publisher-agnostic package discovery by package name alone.
//
// Example:
//
//	key := NameIndexKey("mypackage")
func NameIndexKey(name string) Key {
	input := PrefixNameIndex + name
	return hashToKey(input)
}

// AnnounceKey generates a DHT key for a publisher's announce record.
//
// The key is computed as:
//
//	sha256("libreseed:announce:" + base64(pubkey)) truncated to 20 bytes
//
// The pubkey should be a raw Ed25519 public key (32 bytes).
//
// Example:
//
//	key := AnnounceKey(publisherPubKey)
func AnnounceKey(pubkey []byte) Key {
	encoded := base64.StdEncoding.EncodeToString(pubkey)
	input := PrefixAnnounce + encoded
	return hashToKey(input)
}

// SeederKey generates a DHT key for a seeder's status record.
//
// The key is computed as:
//
//	sha256("libreseed:seeder:" + seederID) truncated to 20 bytes
//
// The seederID is typically base64(sha256(seeder_public_key)).
//
// Example:
//
//	key := SeederKey(seederID)
func SeederKey(seederID string) Key {
	input := PrefixSeeder + seederID
	return hashToKey(input)
}

// GenerateSeederID generates a seeder identity from an Ed25519 public key.
//
// The seeder ID is computed as:
//
//	base64(sha256(seeder_public_key))
//
// This provides cryptographically verifiable seeder identity with no collision risk.
func GenerateSeederID(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// hashToKey computes SHA-256 of the input string and truncates to 20 bytes.
// This produces a key compatible with BitTorrent DHT (which uses 20-byte info hashes).
func hashToKey(input string) Key {
	hash := sha256.Sum256([]byte(input))
	var key Key
	copy(key[:], hash[:DHTKeySize])
	return key
}

// Bytes returns the key as a byte slice.
func (k Key) Bytes() []byte {
	return k[:]
}

// String returns the key as a hex-encoded string.
func (k Key) String() string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, DHTKeySize*2)
	for i, b := range k {
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}
