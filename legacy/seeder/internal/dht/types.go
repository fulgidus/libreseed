// Package dht provides DHT (Distributed Hash Table) functionality for the LibreSeed seeder.
// This file defines the data structures used for DHT record storage and retrieval
// as specified in the LibreSeed protocol specification v1.3.
package dht

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Protocol and version constants as defined in LibreSeed specification v1.3.
const (
	// ProtocolVersion is the current protocol identifier.
	ProtocolVersion = "libreseed-v1"

	// IndexFormatVersion is the current Name Index format version.
	IndexFormatVersion = "1.3"

	// AnnounceFormatVersion is the current Announce format version.
	AnnounceFormatVersion = "1.3"
)

// Field size limits as specified in §7.2 of the protocol specification.
const (
	// MaxPackageNameLength is the maximum length of a package name in bytes.
	MaxPackageNameLength = 64

	// MaxVersionLength is the maximum length of a version string in bytes.
	MaxVersionLength = 32

	// InfohashLength is the length of a BitTorrent v2 infohash (64 hex characters).
	InfohashLength = 64

	// Ed25519PubkeyBase64Length is the approximate base64 length of an Ed25519 public key.
	// Ed25519 public keys are 32 bytes, base64-encoded to ~44 characters.
	Ed25519PubkeyBase64Length = 44

	// Ed25519SignatureBase64Length is the approximate base64 length of an Ed25519 signature.
	// Ed25519 signatures are 64 bytes, base64-encoded to ~88 characters.
	Ed25519SignatureBase64Length = 88

	// MaxPublishersInNameIndex is the recommended maximum number of publishers
	// in a Name Index before local pruning is applied (see §5.3.1).
	MaxPublishersInNameIndex = 300
)

// Validation errors returned by Validate methods.
var (
	ErrEmptyProtocol               = errors.New("protocol field is empty")
	ErrInvalidProtocol             = errors.New("invalid protocol version")
	ErrEmptyName                   = errors.New("name field is empty")
	ErrNameTooLong                 = errors.New("name exceeds maximum length")
	ErrEmptyVersion                = errors.New("version field is empty")
	ErrVersionTooLong              = errors.New("version exceeds maximum length")
	ErrInvalidVersion              = errors.New("invalid version format")
	ErrEmptyInfohash               = errors.New("infohash field is empty")
	ErrInvalidInfohash             = errors.New("invalid infohash format")
	ErrEmptyPubkey                 = errors.New("pubkey field is empty")
	ErrInvalidPubkeyFormat         = errors.New("invalid pubkey format: must be 'ed25519:<hex>'")
	ErrEmptySignature              = errors.New("signature field is empty")
	ErrInvalidSignatureFormat      = errors.New("invalid signature format: must be 'ed25519:<hex>'")
	ErrSignatureVerificationFailed = errors.New("signature verification failed")
	ErrInvalidTimestamp            = errors.New("timestamp is invalid or zero")
	ErrEmptyPublishers             = errors.New("publishers list is empty")
	ErrEmptyPackages               = errors.New("packages list is empty")
	ErrEmptyVersions               = errors.New("versions list is empty")
	ErrEmptyManifestKey            = errors.New("manifestKey field is empty")
	ErrEmptyLatestVersion          = errors.New("latestVersion field is empty")
	ErrEmptyIndexVersion           = errors.New("indexVersion field is empty")
	ErrEmptyAnnounceVersion        = errors.New("announceVersion field is empty")
	ErrEmptySeederID               = errors.New("seederID field is empty")
)

// semverRegex validates semantic version strings (e.g., "1.4.0", "2.0.0-beta.1").
var semverRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)

// hexRegex validates hexadecimal strings.
var hexRegex = regexp.MustCompile(`^[a-fA-F0-9]+$`)

// PublisherSelectionPolicy defines how clients choose among multiple publishers
// for the same package name (see §5.3 and §13.1).
type PublisherSelectionPolicy int

const (
	// PolicyFirstSeen prefers the publisher with the oldest firstSeen timestamp.
	// This is the default policy, providing stability and resistance to name squatting.
	PolicyFirstSeen PublisherSelectionPolicy = iota

	// PolicyLatestVersion prefers the publisher with the highest semantic version.
	// Useful when users prioritize having the newest features.
	PolicyLatestVersion

	// PolicyUserTrust uses a user-defined list of trusted publisher public keys.
	// Falls back to PolicyFirstSeen if no trusted publisher is found.
	PolicyUserTrust

	// PolicySeederCount prefers the publisher with the most active seeders.
	// Useful for maximizing download availability and speed.
	PolicySeederCount
)

// String returns the string representation of a PublisherSelectionPolicy.
func (p PublisherSelectionPolicy) String() string {
	switch p {
	case PolicyFirstSeen:
		return "first-seen"
	case PolicyLatestVersion:
		return "latest-version"
	case PolicyUserTrust:
		return "user-trust"
	case PolicySeederCount:
		return "seeder-count"
	default:
		return fmt.Sprintf("unknown(%d)", int(p))
	}
}

// MinimalManifest represents a version-specific manifest stored in the DHT.
// This is the lightweight record (~500 bytes) that enables package discovery.
// See §7.2 of the protocol specification.
//
// DHT Key: sha256("libreseed:manifest:" + name + "@" + version)
type MinimalManifest struct {
	// Protocol identifies the LibreSeed protocol version (e.g., "libreseed-v1").
	Protocol string `bencode:"protocol" json:"protocol"`

	// Name is the package name (max 64 bytes).
	Name string `bencode:"name" json:"name"`

	// Version is the semantic version string (max 32 bytes).
	Version string `bencode:"version" json:"version"`

	// Infohash is the BitTorrent v2 infohash (64 hex characters).
	// Used to download the torrent containing the full manifest and package contents.
	Infohash string `bencode:"infohash" json:"infohash"`

	// Pubkey is the base64-encoded Ed25519 public key of the publisher.
	Pubkey string `bencode:"pubkey" json:"pubkey"`

	// Signature is the base64-encoded Ed25519 signature over the canonical JSON
	// of all other fields (excluding signature itself).
	Signature string `bencode:"signature" json:"signature"`

	// Timestamp is the Unix timestamp in milliseconds when this manifest was created.
	Timestamp int64 `bencode:"timestamp" json:"timestamp"`
}

// Validate checks that all required fields are present and correctly formatted.
func (m *MinimalManifest) Validate() error {
	if m.Protocol == "" {
		return ErrEmptyProtocol
	}
	if m.Protocol != ProtocolVersion {
		return fmt.Errorf("%w: got %q, expected %q", ErrInvalidProtocol, m.Protocol, ProtocolVersion)
	}
	if m.Name == "" {
		return ErrEmptyName
	}
	if len(m.Name) > MaxPackageNameLength {
		return fmt.Errorf("%w: %d > %d", ErrNameTooLong, len(m.Name), MaxPackageNameLength)
	}
	if m.Version == "" {
		return ErrEmptyVersion
	}
	if len(m.Version) > MaxVersionLength {
		return fmt.Errorf("%w: %d > %d", ErrVersionTooLong, len(m.Version), MaxVersionLength)
	}
	if !semverRegex.MatchString(m.Version) {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, m.Version)
	}
	if m.Infohash == "" {
		return ErrEmptyInfohash
	}
	if len(m.Infohash) != InfohashLength || !hexRegex.MatchString(m.Infohash) {
		return fmt.Errorf("%w: must be %d hex characters", ErrInvalidInfohash, InfohashLength)
	}
	if m.Pubkey == "" {
		return ErrEmptyPubkey
	}
	if m.Signature == "" {
		return ErrEmptySignature
	}
	if m.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

// SigningData returns the canonical JSON bytes used for signature creation/verification.
// The signature field is excluded from the signed data.
func (m *MinimalManifest) SigningData() ([]byte, error) {
	// Create a copy without the signature field for signing
	data := struct {
		Protocol  string `json:"protocol"`
		Name      string `json:"name"`
		Version   string `json:"version"`
		Infohash  string `json:"infohash"`
		Pubkey    string `json:"pubkey"`
		Timestamp int64  `json:"timestamp"`
	}{
		Protocol:  m.Protocol,
		Name:      m.Name,
		Version:   m.Version,
		Infohash:  m.Infohash,
		Pubkey:    m.Pubkey,
		Timestamp: m.Timestamp,
	}
	return canonicalJSON(data)
}

// IsExpired returns true if the manifest timestamp is older than the given TTL.
func (m *MinimalManifest) IsExpired(ttl time.Duration) bool {
	created := time.UnixMilli(m.Timestamp)
	return time.Since(created) > ttl
}

// VerifySignature verifies the Ed25519 signature on the manifest.
// Returns an error if the signature is invalid or verification fails.
func (m *MinimalManifest) VerifySignature() error {
	// Parse pubkey
	pubkeyBytes, err := parseEd25519Key(m.Pubkey)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPubkeyFormat, err)
	}

	// Parse signature
	sigBytes, err := parseEd25519Key(m.Signature)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSignatureFormat, err)
	}

	// Get signing data
	data, err := m.SigningData()
	if err != nil {
		return fmt.Errorf("failed to get signing data: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(ed25519.PublicKey(pubkeyBytes), data, sigBytes) {
		return ErrSignatureVerificationFailed
	}

	return nil
}

// PublisherEntry represents a single publisher's entry in a NameIndex.
// Each entry is independently signed by its publisher (see §5.3).
type PublisherEntry struct {
	// Pubkey is the base64-encoded Ed25519 public key of the publisher.
	Pubkey string `bencode:"pubkey" json:"pubkey"`

	// LatestVersion is the highest version published by this publisher.
	LatestVersion string `bencode:"latestVersion" json:"latestVersion"`

	// FirstSeen is the Unix timestamp in milliseconds when this publisher
	// was first observed for this package name.
	FirstSeen int64 `bencode:"firstSeen" json:"firstSeen"`

	// Signature is the base64-encoded Ed25519 signature over the canonical JSON
	// of: name + latestVersion + firstSeen + timestamp (from parent NameIndex).
	Signature string `bencode:"signature" json:"signature"`
}

// Validate checks that all required fields are present.
func (e *PublisherEntry) Validate() error {
	if e.Pubkey == "" {
		return ErrEmptyPubkey
	}
	if e.LatestVersion == "" {
		return ErrEmptyLatestVersion
	}
	if !semverRegex.MatchString(e.LatestVersion) {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, e.LatestVersion)
	}
	if e.FirstSeen <= 0 {
		return ErrInvalidTimestamp
	}
	if e.Signature == "" {
		return ErrEmptySignature
	}
	return nil
}

// SigningData returns the canonical JSON bytes used for signature creation/verification.
// The name and timestamp parameters come from the parent NameIndex.
func (e *PublisherEntry) SigningData(name string, timestamp int64) ([]byte, error) {
	data := struct {
		Name          string `json:"name"`
		LatestVersion string `json:"latestVersion"`
		FirstSeen     int64  `json:"firstSeen"`
		Timestamp     int64  `json:"timestamp"`
	}{
		Name:          name,
		LatestVersion: e.LatestVersion,
		FirstSeen:     e.FirstSeen,
		Timestamp:     timestamp,
	}
	return canonicalJSON(data)
}

// NameIndex represents a multi-publisher index for a package name.
// It enables publisher-agnostic package discovery (see §5.3).
//
// DHT Key: sha256("libreseed:name-index:" + name)
type NameIndex struct {
	// Protocol identifies the LibreSeed protocol version.
	Protocol string `bencode:"protocol" json:"protocol"`

	// IndexVersion is the Name Index format version (e.g., "1.3").
	IndexVersion string `bencode:"indexVersion" json:"indexVersion"`

	// Name is the package name this index is for.
	Name string `bencode:"name" json:"name"`

	// Publishers is the list of publishers who have published this package.
	// Each entry is independently signed by its respective publisher.
	Publishers []PublisherEntry `bencode:"publishers" json:"publishers"`

	// Timestamp is the Unix timestamp in milliseconds of the last update.
	Timestamp int64 `bencode:"timestamp" json:"timestamp"`
}

// Validate checks that all required fields are present and valid.
func (n *NameIndex) Validate() error {
	if n.Protocol == "" {
		return ErrEmptyProtocol
	}
	if n.Protocol != ProtocolVersion {
		return fmt.Errorf("%w: got %q, expected %q", ErrInvalidProtocol, n.Protocol, ProtocolVersion)
	}
	if n.IndexVersion == "" {
		return ErrEmptyIndexVersion
	}
	if n.Name == "" {
		return ErrEmptyName
	}
	if len(n.Name) > MaxPackageNameLength {
		return fmt.Errorf("%w: %d > %d", ErrNameTooLong, len(n.Name), MaxPackageNameLength)
	}
	if len(n.Publishers) == 0 {
		return ErrEmptyPublishers
	}
	for i, pub := range n.Publishers {
		if err := pub.Validate(); err != nil {
			return fmt.Errorf("publisher[%d]: %w", i, err)
		}
	}
	if n.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

// IsExpired returns true if the index timestamp is older than the given TTL.
func (n *NameIndex) IsExpired(ttl time.Duration) bool {
	updated := time.UnixMilli(n.Timestamp)
	return time.Since(updated) > ttl
}

// FindPublisher returns the PublisherEntry for the given base64-encoded public key,
// or nil if not found.
func (n *NameIndex) FindPublisher(pubkey string) *PublisherEntry {
	for i := range n.Publishers {
		if n.Publishers[i].Pubkey == pubkey {
			return &n.Publishers[i]
		}
	}
	return nil
}

// AnnounceVersion represents a specific version entry within an AnnouncePackage.
// See §6.2 of the protocol specification.
type AnnounceVersion struct {
	// Version is the semantic version string.
	Version string `bencode:"version" json:"version"`

	// ManifestKey is the DHT key for the minimal manifest.
	// Format: sha256("libreseed:manifest:" + name + "@" + version)
	ManifestKey string `bencode:"manifestKey" json:"manifestKey"`

	// Timestamp is the Unix timestamp in milliseconds when this version was published.
	Timestamp int64 `bencode:"timestamp" json:"timestamp"`
}

// Validate checks that all required fields are present.
func (v *AnnounceVersion) Validate() error {
	if v.Version == "" {
		return ErrEmptyVersion
	}
	if !semverRegex.MatchString(v.Version) {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, v.Version)
	}
	if v.ManifestKey == "" {
		return ErrEmptyManifestKey
	}
	if v.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

// IsExpired returns true if the version timestamp is older than the given TTL.
func (v *AnnounceVersion) IsExpired(ttl time.Duration) bool {
	created := time.UnixMilli(v.Timestamp)
	return time.Since(created) > ttl
}

// AnnouncePackage represents a package entry within an Announce record.
// See §6.2 of the protocol specification.
type AnnouncePackage struct {
	// Name is the package name.
	Name string `bencode:"name" json:"name"`

	// LatestVersion is the highest published version of this package.
	LatestVersion string `bencode:"latestVersion" json:"latestVersion"`

	// Versions is the list of all published versions for this package.
	Versions []AnnounceVersion `bencode:"versions" json:"versions"`
}

// Validate checks that all required fields are present and valid.
func (p *AnnouncePackage) Validate() error {
	if p.Name == "" {
		return ErrEmptyName
	}
	if len(p.Name) > MaxPackageNameLength {
		return fmt.Errorf("%w: %d > %d", ErrNameTooLong, len(p.Name), MaxPackageNameLength)
	}
	if p.LatestVersion == "" {
		return ErrEmptyLatestVersion
	}
	if !semverRegex.MatchString(p.LatestVersion) {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, p.LatestVersion)
	}
	if len(p.Versions) == 0 {
		return ErrEmptyVersions
	}
	for i, ver := range p.Versions {
		if err := ver.Validate(); err != nil {
			return fmt.Errorf("version[%d]: %w", i, err)
		}
	}
	return nil
}

// FindVersion returns the AnnounceVersion for the given version string, or nil if not found.
func (p *AnnouncePackage) FindVersion(version string) *AnnounceVersion {
	for i := range p.Versions {
		if p.Versions[i].Version == version {
			return &p.Versions[i]
		}
	}
	return nil
}

// Announce represents a publisher's announce record containing all their packages.
// See §6.2 of the protocol specification.
//
// DHT Key: sha256("libreseed:announce:" + base64(pubkey))
type Announce struct {
	// Protocol identifies the LibreSeed protocol version.
	Protocol string `bencode:"protocol" json:"protocol"`

	// AnnounceVersion is the announce format version (e.g., "1.3").
	AnnounceVersion string `bencode:"announceVersion" json:"announceVersion"`

	// Pubkey is the base64-encoded Ed25519 public key of the publisher.
	Pubkey string `bencode:"pubkey" json:"pubkey"`

	// Timestamp is the Unix timestamp in milliseconds of the last update.
	Timestamp int64 `bencode:"timestamp" json:"timestamp"`

	// Packages is the list of all packages published by this publisher.
	Packages []AnnouncePackage `bencode:"packages" json:"packages"`

	// Signature is the base64-encoded Ed25519 signature over the entire announce
	// document (excluding the signature field itself).
	Signature string `bencode:"signature" json:"signature"`
}

// Validate checks that all required fields are present and valid.
func (a *Announce) Validate() error {
	if a.Protocol == "" {
		return ErrEmptyProtocol
	}
	if a.Protocol != ProtocolVersion {
		return fmt.Errorf("%w: got %q, expected %q", ErrInvalidProtocol, a.Protocol, ProtocolVersion)
	}
	if a.AnnounceVersion == "" {
		return ErrEmptyAnnounceVersion
	}
	if a.Pubkey == "" {
		return ErrEmptyPubkey
	}
	if a.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	if len(a.Packages) == 0 {
		return ErrEmptyPackages
	}
	for i, pkg := range a.Packages {
		if err := pkg.Validate(); err != nil {
			return fmt.Errorf("package[%d]: %w", i, err)
		}
	}
	if a.Signature == "" {
		return ErrEmptySignature
	}
	return nil
}

// SigningData returns the canonical JSON bytes used for signature creation/verification.
// The signature field is excluded from the signed data.
func (a *Announce) SigningData() ([]byte, error) {
	// Create a copy without the signature field for signing
	data := struct {
		Protocol        string            `json:"protocol"`
		AnnounceVersion string            `json:"announceVersion"`
		Pubkey          string            `json:"pubkey"`
		Timestamp       int64             `json:"timestamp"`
		Packages        []AnnouncePackage `json:"packages"`
	}{
		Protocol:        a.Protocol,
		AnnounceVersion: a.AnnounceVersion,
		Pubkey:          a.Pubkey,
		Timestamp:       a.Timestamp,
		Packages:        a.Packages,
	}
	return canonicalJSON(data)
}

// VerifySignature verifies the Ed25519 signature of the Announce using the embedded pubkey.
// The signature is computed over the canonical JSON of all fields except the signature field.
func (a *Announce) VerifySignature() error {
	// Parse public key
	pubkeyBytes, err := parseEd25519Key(a.Pubkey)
	if err != nil {
		return fmt.Errorf("invalid pubkey: %w", err)
	}

	// Parse signature
	signatureBytes, err := parseEd25519Key(a.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Get signing data
	signingData, err := a.SigningData()
	if err != nil {
		return fmt.Errorf("failed to create signing data: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(pubkeyBytes, signingData, signatureBytes) {
		return ErrSignatureVerificationFailed
	}

	return nil
}

// IsExpired returns true if the announce timestamp is older than the given TTL.
func (a *Announce) IsExpired(ttl time.Duration) bool {
	updated := time.UnixMilli(a.Timestamp)
	return time.Since(updated) > ttl
}

// FindPackage returns the AnnouncePackage for the given package name, or nil if not found.
func (a *Announce) FindPackage(name string) *AnnouncePackage {
	for i := range a.Packages {
		if a.Packages[i].Name == name {
			return &a.Packages[i]
		}
	}
	return nil
}

// SeederStatus represents a seeder's status record in the DHT.
// See §5.2 of the protocol specification.
//
// DHT Key: sha256("libreseed:seeder:" + seederID)
// where seederID = base64(sha256(seeder_public_key))
type SeederStatus struct {
	// Protocol identifies the LibreSeed protocol version.
	Protocol string `bencode:"protocol" json:"protocol"`

	// SeederID is the unique identifier for this seeder.
	// Format: base64(sha256(seeder_public_key))
	SeederID string `bencode:"seederID" json:"seederID"`

	// Pubkey is the base64-encoded Ed25519 public key of the seeder.
	Pubkey string `bencode:"pubkey" json:"pubkey"`

	// Timestamp is the Unix timestamp in milliseconds of the last status update.
	Timestamp int64 `bencode:"timestamp" json:"timestamp"`

	// SeededPackages is the list of package identifiers currently being seeded.
	// Format: "name@version" for each package.
	SeededPackages []string `bencode:"seededPackages" json:"seededPackages"`

	// UptimeSeconds is the number of seconds the seeder has been running.
	UptimeSeconds int64 `bencode:"uptimeSeconds" json:"uptimeSeconds"`

	// DiskUsageBytes is the total disk space used for seeded packages.
	DiskUsageBytes int64 `bencode:"diskUsageBytes" json:"diskUsageBytes"`

	// BandwidthStats contains upload/download statistics.
	BandwidthStats BandwidthStats `bencode:"bandwidthStats" json:"bandwidthStats"`

	// Signature is the base64-encoded Ed25519 signature over the entire status
	// document (excluding the signature field itself).
	Signature string `bencode:"signature" json:"signature"`
}

// BandwidthStats contains network bandwidth statistics for a seeder.
type BandwidthStats struct {
	// TotalUploadBytes is the total bytes uploaded since seeder start.
	TotalUploadBytes int64 `bencode:"totalUploadBytes" json:"totalUploadBytes"`

	// TotalDownloadBytes is the total bytes downloaded since seeder start.
	TotalDownloadBytes int64 `bencode:"totalDownloadBytes" json:"totalDownloadBytes"`

	// CurrentUploadRate is the current upload rate in bytes per second.
	CurrentUploadRate int64 `bencode:"currentUploadRate" json:"currentUploadRate"`

	// CurrentDownloadRate is the current download rate in bytes per second.
	CurrentDownloadRate int64 `bencode:"currentDownloadRate" json:"currentDownloadRate"`
}

// Validate checks that all required fields are present and valid.
func (s *SeederStatus) Validate() error {
	if s.Protocol == "" {
		return ErrEmptyProtocol
	}
	if s.Protocol != ProtocolVersion {
		return fmt.Errorf("%w: got %q, expected %q", ErrInvalidProtocol, s.Protocol, ProtocolVersion)
	}
	if s.SeederID == "" {
		return ErrEmptySeederID
	}
	if s.Pubkey == "" {
		return ErrEmptyPubkey
	}
	if s.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	if s.Signature == "" {
		return ErrEmptySignature
	}
	return nil
}

// SigningData returns the canonical JSON bytes used for signature creation/verification.
// The signature field is excluded from the signed data.
func (s *SeederStatus) SigningData() ([]byte, error) {
	// Create a copy without the signature field for signing
	data := struct {
		Protocol       string         `json:"protocol"`
		SeederID       string         `json:"seederID"`
		Pubkey         string         `json:"pubkey"`
		Timestamp      int64          `json:"timestamp"`
		SeededPackages []string       `json:"seededPackages"`
		UptimeSeconds  int64          `json:"uptimeSeconds"`
		DiskUsageBytes int64          `json:"diskUsageBytes"`
		BandwidthStats BandwidthStats `json:"bandwidthStats"`
	}{
		Protocol:       s.Protocol,
		SeederID:       s.SeederID,
		Pubkey:         s.Pubkey,
		Timestamp:      s.Timestamp,
		SeededPackages: s.SeededPackages,
		UptimeSeconds:  s.UptimeSeconds,
		DiskUsageBytes: s.DiskUsageBytes,
		BandwidthStats: s.BandwidthStats,
	}
	return canonicalJSON(data)
}

// VerifySignature verifies the Ed25519 signature of the SeederStatus using the embedded pubkey.
// The signature is computed over the canonical JSON of all fields except the signature field.
func (s *SeederStatus) VerifySignature() error {
	// Parse public key
	pubkeyBytes, err := parseEd25519Key(s.Pubkey)
	if err != nil {
		return fmt.Errorf("invalid pubkey: %w", err)
	}

	// Parse signature
	signatureBytes, err := parseEd25519Key(s.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Get signing data
	signingData, err := s.SigningData()
	if err != nil {
		return fmt.Errorf("failed to create signing data: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(pubkeyBytes, signingData, signatureBytes) {
		return ErrSignatureVerificationFailed
	}

	return nil
}

// IsExpired returns true if the status timestamp is older than the given TTL.
func (s *SeederStatus) IsExpired(ttl time.Duration) bool {
	updated := time.UnixMilli(s.Timestamp)
	return time.Since(updated) > ttl
}

// canonicalJSON produces deterministic JSON output with sorted keys.
// This is required for consistent signature generation and verification.
func canonicalJSON(v interface{}) ([]byte, error) {
	// json.Marshal produces deterministic output for struct types
	// (fields are output in definition order)
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("canonical JSON marshal failed: %w", err)
	}

	// Compact the JSON to remove any whitespace
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, fmt.Errorf("canonical JSON compact failed: %w", err)
	}

	return buf.Bytes(), nil
}

// parseEd25519Key parses an Ed25519 key or signature in the format "ed25519:<hex>".
// Returns the decoded bytes or an error if the format is invalid.
// Accepts 32 bytes (public key) or 64 bytes (signature).
func parseEd25519Key(s string) ([]byte, error) {
	// Check for "ed25519:" prefix
	const prefix = "ed25519:"
	if !strings.HasPrefix(s, prefix) {
		return nil, fmt.Errorf("missing 'ed25519:' prefix")
	}

	// Extract hex string after prefix
	hexStr := strings.TrimPrefix(s, prefix)
	if hexStr == "" {
		return nil, fmt.Errorf("empty hex string after prefix")
	}

	// Decode hex
	keyBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}

	// Validate length: must be 32 bytes (public key) or 64 bytes (signature)
	if len(keyBytes) != ed25519.PublicKeySize && len(keyBytes) != ed25519.SignatureSize {
		return nil, fmt.Errorf("invalid key length: got %d bytes, expected %d (public key) or %d (signature)",
			len(keyBytes), ed25519.PublicKeySize, ed25519.SignatureSize)
	}

	return keyBytes, nil
}

// encodeEd25519Key encodes an Ed25519 key or signature as "ed25519:<hex>".
func encodeEd25519Key(keyBytes []byte) string {
	return "ed25519:" + hex.EncodeToString(keyBytes)
}
