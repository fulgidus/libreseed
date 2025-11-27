# LibreSeed CLI Documentation

Command-line interface for LibreSeed operations.

## Overview

The LibreSeed CLI provides commands for:
- Publishing packages
- Verifying package signatures
- Searching for packages in DHT
- Managing keypairs
- Seeder operations

## Installation

See [INSTALLATION.md](INSTALLATION.md) for complete installation guide.

### Quick Install

```bash
# Linux/macOS
curl -sSL https://get.libreseed.org | sh

# Manual download
wget https://github.com/libreseed/libreseed/releases/download/v1.0.0/libreseed-$(uname -s)-$(uname -m)
chmod +x libreseed-*
sudo mv libreseed-* /usr/local/bin/libreseed
```

### Verify Installation

```bash
libreseed version
```

## Command Reference

See [COMMANDS.md](COMMANDS.md) for complete command reference.

### Core Commands

#### `libreseed publish`

Publish a package to LibreSeed:

```bash
libreseed publish [options]

Options:
  --name <name>           Package name
  --version <version>     Package version
  --key <path>            Private key file
  --tarball <path>        Package tarball
  --seeder <url>          Seeder URL for file upload
  --manifest <path>       Pre-built manifest file
  --description <text>    Package description
  --homepage <url>        Homepage URL
```

**Example:**

```bash
libreseed publish \
  --name mypackage \
  --version 1.0.0 \
  --key ~/.libreseed/key.pem \
  --tarball mypackage-1.0.0.tgz \
  --seeder https://seed1.example.com
```

#### `libreseed keygen`

Generate Ed25519 keypair:

```bash
libreseed keygen [options]

Options:
  --output <path>         Output file path (default: libreseed-key.pem)
  --format <format>       Key format: pem, json (default: pem)
  --passphrase <pass>     Encrypt with passphrase
```

**Example:**

```bash
libreseed keygen --output publisher-key.pem --passphrase
```

#### `libreseed verify`

Verify package signature:

```bash
libreseed verify [options]

Options:
  --name <name>           Package name
  --version <version>     Package version
  --publisher <pubkey>    Publisher public key
  --manifest <path>       Manifest file to verify
```

**Example:**

```bash
libreseed verify \
  --name mypackage \
  --version 1.0.0 \
  --publisher ABC123...
```

#### `libreseed search`

Search for packages in DHT:

```bash
libreseed search [options]

Options:
  --name <name>           Package name
  --version <version>     Package version (optional)
  --publisher <pubkey>    Filter by publisher
  --timeout <duration>    Search timeout (default: 30s)
```

**Example:**

```bash
libreseed search --name mypackage
```

### Seeder Commands

#### `libreseed seeder start`

Start a seeder node:

```bash
libreseed seeder start [options]

Options:
  --config <path>         Config file path (default: seeder.yaml)
  --data-dir <path>       Data directory
  --port <port>           DHT port (default: 6881)
  --http-port <port>      HTTP API port (default: 8080)
```

#### `libreseed seeder status`

Check seeder status:

```bash
libreseed seeder status [options]

Options:
  --url <url>             Seeder API URL
```

### Utility Commands

#### `libreseed inspect`

Inspect manifest file:

```bash
libreseed inspect <manifest-file>
```

#### `libreseed download`

Download package from DHT:

```bash
libreseed download [options]

Options:
  --name <name>           Package name
  --version <version>     Package version
  --output <dir>          Output directory
```

## Configuration

CLI can be configured via:

1. Command-line flags
2. Environment variables
3. Configuration file

### Configuration File

Create `~/.config/libreseed/config.yaml`:

```yaml
default_seeder: https://seed1.example.com
publisher_key: ~/.libreseed/key.pem
dht_bootstrap:
  - router.bittorrent.com:6881
  - dht.transmissionbt.com:6881
```

### Environment Variables

```bash
export LIBRESEED_KEY=~/.libreseed/key.pem
export LIBRESEED_SEEDER=https://seed1.example.com
export LIBRESEED_DHT_PORT=6881
```

## Output Formats

CLI supports multiple output formats:

```bash
# JSON output
libreseed search --name mypackage --format json

# YAML output
libreseed search --name mypackage --format yaml

# Table output (default)
libreseed search --name mypackage --format table
```

## Debugging

Enable debug logging:

```bash
export LIBRESEED_DEBUG=1
libreseed publish ...
```

Verbose output:

```bash
libreseed --verbose publish ...
```

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Network error
- `4` - Signature verification failed
- `5` - DHT operation failed

## Related Documentation

- [Publisher Documentation](../publisher/) - Publishing workflow
- [Seeder Documentation](../seeder/) - Running a seeder
- [Protocol Specification](../spec/LIBRESEED-SPEC-v1.2.md)
