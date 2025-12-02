# LibreSeed Seeder - Architectural Decision Records

**Version:** 1.0  
**Status:** Draft  
**Last Updated:** 2025-01-27  
**Language:** en-US

---

## ADR-001: Choice of anacrolix/torrent Library

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

The LibreSeed Seeder requires a robust BitTorrent implementation in Go with DHT support. Several options were evaluated:

1. **anacrolix/torrent**: Mature, feature-complete Go BitTorrent library with DHT
2. **jackpal/go-nat-pmp + custom implementation**: Build from scratch
3. **marksamman/bencode + custom implementation**: Low-level approach
4. **CGo bindings to libtorrent**: C++ library via CGo

### Decision

**Selected**: `anacrolix/torrent`

### Rationale

**Pros**:
- **Mature and Maintained**: 8+ years of active development, 5.4k+ GitHub stars
- **Feature Complete**: Supports BitTorrent v1/v2, DHT, encryption, uTP, peer exchange
- **DHT Built-in**: Mainline DHT implementation with extensibility for custom keys
- **Pure Go**: No CGo dependencies, easy cross-compilation
- **Performance**: Proven in production environments (used by several large-scale projects)
- **Active Community**: Regular updates, responsive maintainers
- **Documentation**: Comprehensive godoc and examples

**Cons**:
- **Large Dependency**: Brings in many sub-packages (~20+ dependencies)
- **API Complexity**: Advanced features require understanding internal architecture
- **Customization Overhead**: LibreSeed's custom DHT keys require wrapper layer

**Alternatives Rejected**:
- **Custom Implementation**: Too risky, requires 6+ months of development and testing
- **CGo + libtorrent**: Cross-compilation complexity, performance overhead
- **Low-level Libraries**: Missing critical features like DHT, would require extensive development

### Consequences

- Rapid development: Weeks instead of months
- Stable foundation: Leverage battle-tested implementation
- Maintenance burden: Must track upstream changes
- Customization layer: Need wrapper for LibreSeed DHT keys (~500 LOC)

### Implementation Notes

**Custom DHT Layer**:
```go
// Wrap anacrolix/torrent DHT with LibreSeed key scheme
type LibreSeedDHT struct {
    client *torrent.Client
}

func (d *LibreSeedDHT) AnnouncePackage(pkg string, infohash string) error {
    key := dht.MakeSHA1Hash("libreseed:pkg:" + pkg)
    return d.client.DHT().Announce(key, infohash)
}
```

**References**:
- [anacrolix/torrent GitHub](https://github.com/anacrolix/torrent)
- [anacrolix/torrent Documentation](https://pkg.go.dev/github.com/anacrolix/torrent)

---

## ADR-002: DHT Customization Strategy

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

LibreSeed uses custom DHT keys (`libreseed:pkg:<name>`) instead of standard infohash-only announces. This requires customization of the DHT layer while maintaining BitTorrent compatibility.

**LibreSeed DHT Keys**:
- `libreseed:pkg:<name>` → Latest version manifest (mutable)
- `libreseed:ver:<name>:<version>` → Version-specific manifest (immutable)
- `libreseed:announce:<pubkey>` → Publisher's package list

**Challenge**: Standard DHT expects 20-byte infohashes, not arbitrary strings.

### Decision

**Approach**: **SHA-1 Hash Mapping with Dual Announces**

1. Hash LibreSeed keys to 20-byte DHT keys via SHA-1
2. Store minimal manifest (~500 bytes) in DHT value
3. Announce **both** LibreSeed keys **and** standard infohash
4. Maintain separate announce timers (22 hours for LibreSeed, default for standard)

### Rationale

**SHA-1 Hash Mapping**:
```go
dhtKey := sha1.Sum([]byte("libreseed:pkg:example-package"))
// dhtKey is now a valid 20-byte DHT key
```

**Dual Announce Strategy**:
- **Standard Infohash Announce**: Enables standard BitTorrent clients to participate
- **LibreSeed Key Announce**: Enables package discovery without knowing infohash

**Minimal Manifest Design**:
```json
{
  "infohash": "abc123...",
  "size": 1024000,
  "timestamp": 1706356800,
  "v": 1
}
```
Size: ~120-150 bytes (fits comfortably in DHT value limit of ~1KB)

**Alternatives Considered**:
1. **Fork anacrolix/torrent DHT**: Too invasive, hard to maintain
2. **Separate Custom DHT**: Fragments ecosystem, reinvents wheel
3. **No LibreSeed Keys**: Loses decentralized discovery feature

### Consequences

**Positive**:
- Standard BitTorrent compatibility maintained
- LibreSeed discovery works as designed
- Minimal code changes (~300 LOC wrapper)

**Negative**:
- Double announce overhead (minimal, only every 22 hours)
- SHA-1 collision risk (negligible for package names)
- Key squatting possible (no central authority)

### Implementation Notes

**Announce Workflow**:
```go
func (d *DHTManager) AnnouncePackage(pkg PackageInfo) error {
    // 1. Announce standard infohash (for BitTorrent compatibility)
    err := d.announceInfohash(pkg.Infohash)
    
    // 2. Announce libreseed:pkg:<name> (for discovery)
    pkgKey := makeDHTKey("libreseed:pkg:" + pkg.Name)
    manifest := createMinimalManifest(pkg)
    err = d.announceMutable(pkgKey, manifest)
    
    // 3. Announce libreseed:ver:<name>:<version> (immutable)
    verKey := makeDHTKey("libreseed:ver:" + pkg.Name + ":" + pkg.Version)
    err = d.announceImmutable(verKey, manifest)
    
    return err
}
```

**Re-announce Timer**: 22 hours as per LibreSeed spec §3.2

---

## ADR-003: Folder Watching Mechanism

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

LibreSeed's shared folder workflow requires automatic detection of `.torrent` files dropped by publishers. The seeder must:
- Detect new `.torrent` files in real-time
- Validate files before adding
- Handle high-frequency additions (100+ files/minute)
- Support multiple watch directories

**Options**:
1. **Polling**: Check directory every N seconds
2. **fsnotify**: Filesystem event notifications
3. **inotify (Linux-only)**: Direct Linux kernel interface
4. **File Watcher Services**: Third-party cloud-based solutions

### Decision

**Selected**: `github.com/fsnotify/fsnotify`

### Rationale

**Pros**:
- **Cross-Platform**: Works on Linux, macOS, Windows, BSD
- **Real-Time**: Sub-second detection via kernel events (inotify, FSEvents, etc.)
- **Low Overhead**: Event-driven, no polling CPU waste
- **Mature**: Stable API, 9+ years of development
- **Pure Go**: No CGo dependencies

**Cons**:
- **Event Complexity**: Need to handle CREATE, WRITE, RENAME correctly
- **Debouncing Required**: Large files trigger multiple WRITE events
- **Recursive Watching**: Must manually watch subdirectories

**Implementation Strategy**:
```go
watcher, _ := fsnotify.NewWatcher()
watcher.Add("/path/to/torrents")

for {
    select {
    case event := <-watcher.Events:
        if event.Op&fsnotify.Create == fsnotify.Create {
            if strings.HasSuffix(event.Name, ".torrent") {
                // Debounce: wait 1 second for WRITE to complete
                time.Sleep(1 * time.Second)
                addTorrent(event.Name)
            }
        }
    case err := <-watcher.Errors:
        log.Error(err)
    }
}
```

**Debouncing Strategy**:
- Detect `.torrent` CREATE event
- Wait 1 second for file writes to complete
- Validate file integrity (bencode parsing)
- Add to torrent engine

**Alternatives Rejected**:
- **Polling**: 5-60 second delay, wasted CPU cycles
- **inotify Direct**: Linux-only, breaks cross-platform goal
- **Cloud Services**: Requires internet, adds complexity

### Consequences

- **Real-time responsiveness**: Sub-second `.torrent` detection
- **Cross-platform**: Works on all LibreSeed target platforms
- **Complexity**: Need to handle edge cases (rapid file operations, renames, deletes)
- **Testing**: Requires filesystem-level integration tests

### Configuration

```yaml
folder_watcher:
  enabled: true
  paths:
    - ~/.libreseed/torrents/
    - /mnt/shared/torrents/
  debounce_ms: 1000
  auto_add: true
  archive_added: true
  archive_path: ~/.libreseed/torrents/archive/
```

---

## ADR-004: Storage Management Strategy

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

The Seeder must manage disk storage efficiently:
- Prevent disk exhaustion
- Support configurable storage quotas
- Allow dynamic torrent prioritization
- Handle cleanup of old/orphaned files

**Challenges**:
- Users may have limited disk space
- Torrents can be 10MB to 100GB+
- Need to balance seeding vs. disk constraints

### Decision

**Strategy**: **Soft Quota with Configurable Limits**

**Implementation**:
1. **Configurable Max Size**: `max_storage: 100GB` (0 = unlimited)
2. **Reserved Space**: Always keep N GB free (default: 5GB)
3. **Auto-Cleanup**: Remove oldest torrents when quota exceeded (opt-in)
4. **Priority System**: Tag torrents as "permanent" or "ephemeral"

### Rationale

**Soft Quota Benefits**:
- User control: Explicit configuration, no surprises
- Flexibility: Can exceed quota temporarily if space available
- Graceful degradation: Warning before hard stop

**Reserved Space**:
```go
func (s *StorageManager) CanFitTorrent(size int64) bool {
    available := s.GetAvailableSpace()
    reserved := s.config.ReservedSpace
    maxSize := s.config.MaxSize
    
    if maxSize > 0 && s.GetUsedSpace() + size > maxSize {
        return false
    }
    
    return available - size > reserved
}
```

**Auto-Cleanup Strategy**:
1. Triggered when `CanFitTorrent()` returns false
2. Remove oldest torrents marked "ephemeral"
3. Never remove "permanent" torrents
4. Log all cleanup actions

**Alternatives Considered**:
- **Hard Quota**: Too rigid, fails on quota exceeded
- **No Limits**: Risk disk exhaustion, crashes
- **LRU Cache**: Complex, hard to predict behavior

### Consequences

**Positive**:
- Predictable disk usage
- User control over limits
- Prevents accidental disk fills

**Negative**:
- Cleanup logic adds complexity
- Users must configure limits

### Configuration

```yaml
storage:
  data_dir: ~/.libreseed/data/
  max_size: 107374182400  # 100 GB (0 = unlimited)
  reserved_space: 5368709120  # 5 GB
  auto_cleanup: true
  cleanup_policy: oldest_first  # oldest_first, lowest_ratio, largest_size
```

---

## ADR-005: Health Monitoring Approach

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

Operators need visibility into seeder health:
- Is the seeder running?
- How many torrents are active?
- Upload/download rates?
- DHT connectivity?
- Disk usage?

**Requirements**:
- Real-time metrics
- Prometheus integration
- Structured logging
- Minimal performance overhead

### Decision

**Approach**: **Prometheus Metrics + Structured JSON Logs**

**Metrics**:
- Expose `/metrics` endpoint for Prometheus scraping
- Use `prometheus/client_golang`
- Update metrics on-demand (no background polling)

**Logging**:
- Structured JSON logs via `uber-go/zap` (high performance)
- Log levels: DEBUG, INFO, WARN, ERROR
- Contextual fields: component, infohash, package name

### Rationale

**Prometheus Advantages**:
- Industry standard for metrics
- Easy integration with Grafana
- Built-in alerting (Alertmanager)
- Pull-based model (no metric forwarding)

**Zap Logging Advantages**:
- High performance (10x faster than logrus)
- Zero-allocation JSON encoding
- Structured context via `zap.Field`

**Alternatives Considered**:
- **StatsD**: Push-based, requires additional daemon
- **Custom Metrics API**: Reinvents wheel
- **Plain Text Logs**: Hard to parse, no structure

### Consequences

**Positive**:
- Industry-standard observability
- Easy integration with monitoring stacks
- High performance logging

**Negative**:
- Requires Prometheus for full value
- JSON logs not human-readable (use `jq` or log viewer)

### Implementation

**Metrics Example**:
```go
var (
    torrentsActive = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "libreseed_torrents_active",
        Help: "Number of active torrents",
    })
    
    uploadBytesTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "libreseed_upload_bytes_total",
        Help: "Total bytes uploaded",
    })
)

func (h *HealthMonitor) RecordUpload(bytes int64) {
    uploadBytesTotal.Add(float64(bytes))
}
```

**Log Example**:
```go
logger.Info("Torrent added",
    zap.String("component", "core-engine"),
    zap.String("infohash", infohash),
    zap.String("name", name),
    zap.Int64("size", size),
)
```

### Configuration

```yaml
monitoring:
  metrics_enabled: true
  metrics_addr: ":9090"
  log_level: info  # debug, info, warn, error
  log_format: json  # json, console
  log_file: ~/.libreseed/logs/seeder.log
```

---

## ADR-006: CLI Framework Choice

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

The Seeder requires a command-line interface for:
- Starting/stopping the daemon
- Adding/removing torrents
- Viewing status
- Managing configuration

**Requirements**:
- Subcommands (`start`, `stop`, `add`, `remove`, `list`, `status`)
- Flags and options (`--config`, `--verbose`, `--daemon`)
- Help text generation
- Bash/Zsh completion (nice-to-have)

**Options**:
1. **spf13/cobra**: Feature-rich CLI framework
2. **urfave/cli**: Simpler alternative
3. **flag (stdlib)**: Minimal, manual parsing
4. **alecthomas/kong**: Struct-based CLI definition

### Decision

**Selected**: `spf13/cobra`

### Rationale

**Pros**:
- **Industry Standard**: Used by kubectl, Hugo, Docker CLI, GitHub CLI
- **Feature Complete**: Subcommands, flags, aliases, help generation, completion
- **Integration**: Works seamlessly with `spf13/viper` for configuration
- **Documentation**: Excellent docs and examples
- **Maintenance**: Active development, used by thousands of projects

**Cons**:
- **Complexity**: More features than needed for simple CLIs
- **Dependency Size**: Larger than minimal alternatives

**Command Structure**:
```
libreseed-seeder start [--config FILE] [--daemon]
libreseed-seeder stop
libreseed-seeder add <torrent-file>
libreseed-seeder remove <infohash>
libreseed-seeder list [--verbose]
libreseed-seeder status
libreseed-seeder config validate [--config FILE]
libreseed-seeder version
```

**Alternatives Rejected**:
- **urfave/cli**: Less feature-rich, different API style
- **flag stdlib**: Too manual, no subcommand support
- **kong**: Struct tags verbose for complex CLIs

### Consequences

**Positive**:
- Professional CLI UX matching industry standards
- Auto-generated help text
- Easy to extend with new commands
- Built-in shell completion

**Negative**:
- Larger binary size (~2MB compiled)
- More code to learn for contributors

### Implementation Example

```go
var rootCmd = &cobra.Command{
    Use:   "libreseed-seeder",
    Short: "LibreSeed package seeder daemon",
}

var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the seeder daemon",
    Run: func(cmd *cobra.Command, args []string) {
        // Start daemon
    },
}

func init() {
    startCmd.Flags().StringP("config", "c", "", "Config file path")
    startCmd.Flags().BoolP("daemon", "d", false, "Run in background")
    rootCmd.AddCommand(startCmd)
}
```

---

## ADR-007: Configuration File Format

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

The Seeder requires configuration for:
- Network settings (ports, bandwidth limits)
- Storage paths and quotas
- DHT parameters
- Folder watch paths
- Monitoring endpoints

**Requirements**:
- Human-readable and editable
- Comments support
- Hierarchical structure
- Validation support

**Options**:
1. **YAML**: Human-readable, widely used
2. **TOML**: Simpler than YAML, clear syntax
3. **JSON**: Ubiquitous, no comments
4. **INI**: Simple but limited

### Decision

**Selected**: **YAML**

### Rationale

**Pros**:
- **Readability**: Clean syntax with minimal punctuation
- **Comments**: Full comment support for documentation
- **Complex Structures**: Nested maps, arrays, multi-line strings
- **Ecosystem**: Wide tool support (editors, validators, converters)
- **Library Support**: `spf13/viper` has excellent YAML support

**Cons**:
- **Indentation Sensitive**: Whitespace errors possible
- **Complexity**: More features than needed (anchors, aliases)

**Example Configuration**:
```yaml
# LibreSeed Seeder Configuration

network:
  listen_addr: "0.0.0.0:6881"
  max_upload_rate: 10485760  # 10 MB/s (0 = unlimited)
  max_download_rate: 5242880  # 5 MB/s
  max_peers_per_torrent: 50
  enable_dht: true
  enable_pex: true
  enable_upnp: false

storage:
  data_dir: ~/.libreseed/data/
  max_size: 107374182400  # 100 GB
  reserved_space: 5368709120  # 5 GB
  auto_cleanup: true

folder_watcher:
  enabled: true
  paths:
    - ~/.libreseed/torrents/
  debounce_ms: 1000
  auto_add: true
  archive_added: true

dht:
  bootstrap_nodes:
    - router.bittorrent.com:6881
    - dht.transmissionbt.com:6881
  reannounce_interval: 79200  # 22 hours in seconds

monitoring:
  metrics_enabled: true
  metrics_addr: ":9090"
  log_level: info
  log_format: json

api:
  enabled: true
  listen_addr: ":8080"
  auth_token: ""  # Optional Bearer token
```

**Alternatives Rejected**:
- **TOML**: Less popular in Go ecosystem
- **JSON**: No comments, harder to edit manually
- **INI**: Too limited for nested structures

### Consequences

**Positive**:
- Familiar format for most users
- Easy to document inline with comments
- `viper` provides excellent parsing and validation

**Negative**:
- Indentation errors can be frustrating
- Multiple ways to express same thing (can confuse users)

### Validation

Use `viper` for loading and validation:
```go
func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    
    return &cfg, cfg.Validate()
}

func (c *Config) Validate() error {
    if c.Storage.MaxSize < 0 {
        return errors.New("storage.max_size cannot be negative")
    }
    // ... more validations
    return nil
}
```

---

## ADR-008: BitTorrent Compatibility Approach

**Status**: Accepted  
**Date**: 2025-01-27  
**Deciders**: Architecture Team

### Context

LibreSeed must maintain compatibility with standard BitTorrent clients while adding custom DHT discovery. Key question: Should we require protocol modifications in standard clients?

**Options**:
1. **Full Compatibility**: Standard clients work without modification
2. **LibreSeed Extension**: Require BEP extension in clients
3. **Proprietary Protocol**: Fork BitTorrent protocol entirely

### Decision

**Selected**: **Full Compatibility with Optional LibreSeed Extensions**

**Design**:
- Standard BitTorrent clients can seed **any** LibreSeed package (given `.torrent` file)
- LibreSeed-aware clients gain additional discovery features
- No protocol modifications required for basic functionality

### Rationale

**Full Compatibility Benefits**:
- Leverage existing BitTorrent ecosystem (millions of clients)
- No adoption barriers for seeders
- Gradual migration path for LibreSeed features

**How It Works**:
1. LibreSeed Seeder announces **standard infohash** to DHT (BEP-5)
2. Standard clients find peers via standard DHT lookup
3. Swarm participation uses standard BitTorrent protocol
4. LibreSeed keys (`libreseed:pkg:<name>`) are **additional**, not required

**Example Scenario**:
```
1. LibreSeed Publisher creates example-package-v1.0.0.torrent
2. Publisher uploads .torrent to HTTP server
3. User downloads .torrent via browser
4. User opens .torrent in qBittorrent
5. qBittorrent performs standard infohash DHT lookup
6. qBittorrent finds LibreSeed Seeder (and other peers) in swarm
7. qBittorrent seeds/downloads normally
```

**LibreSeed Extensions (Optional)**:
- **Discovery**: LibreSeed-aware clients can find packages by name (no `.torrent` needed)
- **Version Queries**: Query specific versions via DHT
- **Publisher Tracking**: Follow publisher's package announcements

**Alternatives Rejected**:
- **LibreSeed-Only**: Fragments ecosystem, limits adoption
- **Protocol Fork**: Incompatible with existing clients, no network effects

### Consequences

**Positive**:
- Maximum compatibility and adoption potential
- Leverage existing BitTorrent infrastructure
- Lower barrier to entry for seeders

**Negative**:
- Standard clients lack discovery features
- Need to distribute `.torrent` files via other means (HTTP, shared folders)

### Implementation

**Seeder Announces**:
```go
// 1. Standard infohash announce (for compatibility)
dht.Announce(infohash, port)

// 2. LibreSeed package announce (for discovery)
pkgKey := sha1("libreseed:pkg:" + name)
dht.Put(pkgKey, minimalManifest)

// 3. LibreSeed version announce
verKey := sha1("libreseed:ver:" + name + ":" + version)
dht.Put(verKey, minimalManifest)
```

Both announces happen simultaneously. Standard clients use #1, LibreSeed clients can use #2 or #3.

---

## Summary Table

| ADR | Decision | Rationale |
|-----|----------|-----------|
| ADR-001 | Use `anacrolix/torrent` | Mature, feature-complete, pure Go BitTorrent library |
| ADR-002 | SHA-1 hash DHT keys with dual announces | Maintains BitTorrent compatibility while enabling LibreSeed discovery |
| ADR-003 | Use `fsnotify` for folder watching | Cross-platform, real-time, low overhead |
| ADR-004 | Soft quota with configurable limits | Predictable disk usage, user control, graceful degradation |
| ADR-005 | Prometheus + Zap structured logging | Industry standard, high performance, easy integration |
| ADR-006 | Use `cobra` CLI framework | Professional UX, feature-complete, industry standard |
| ADR-007 | YAML configuration format | Human-readable, comments, hierarchical, widely supported |
| ADR-008 | Full BitTorrent compatibility | Maximum adoption, leverage existing ecosystem |

---

## Revision History

| Version | Date       | Author | Changes |
|---------|------------|--------|---------|
| 1.0     | 2025-01-27 | Architecture Team | Initial ADR document with 8 decisions |
