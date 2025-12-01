# LibreSeed NPM Gateway Documentation

The NPM Gateway integrates LibreSeed with the npm package manager.

## Overview

The NPM Gateway allows npm projects to:
- Install packages directly from LibreSeed DHT
- Fall back to npm registry when packages not in LibreSeed
- Use familiar npm commands (`npm install`, `npm update`)
- Maintain compatibility with existing npm workflows

## Quick Start

### 1. Install Gateway

```bash
npm install -g libreseed-gateway-npm
```

### 2. Configure npm Project

Add to `package.json`:

```json
{
  "name": "my-project",
  "dependencies": {
    "some-package": "^1.0.0"
  },
  "libreseed": {
    "enabled": true,
    "packages": {
      "some-package": {
        "source": "libreseed",
        "publisher": "ABC123..."
      }
    }
  }
}
```

### 3. Install Dependencies

```bash
npm install
```

The gateway will:
1. Check if `some-package` is configured for LibreSeed
2. Query DHT for `libreseed:manifest:some-package@1.0.0`
3. Verify signature matches configured publisher
4. Download and extract package
5. Create symlink in `node_modules/`

## Documentation

- **[FEASIBILITY-ANALYSIS.md](FEASIBILITY-ANALYSIS.md)** - Technical analysis of module resolution
- **[INTEGRATION.md](INTEGRATION.md)** - `package.json` configuration guide
- **[MODULE-RESOLUTION.md](MODULE-RESOLUTION.md)** - How symlink strategy works

## Architecture

### Components

1. **Gateway CLI** - npm wrapper intercepting install commands
2. **DHT Client** - Queries LibreSeed DHT for manifests
3. **Signature Verifier** - Verifies Ed25519 signatures
4. **Package Downloader** - Downloads files from seeders
5. **Symlink Manager** - Creates/manages `node_modules` symlinks

### Workflow

```
npm install
    ↓
Gateway intercepts
    ↓
Check package.json libreseed config
    ↓
Query DHT for manifest
    ↓
Verify signature
    ↓
Download package files
    ↓
Extract to .libreseed/cache/
    ↓
Symlink to node_modules/
    ↓
Continue npm install for remaining packages
```

## Configuration

### Global Configuration

Create `~/.libreseed/npm-gateway.yaml`:

```yaml
# DHT configuration
dht:
  bootstrap_nodes:
    - router.bittorrent.com:6881
    - dht.transmissionbt.com:6881
  timeout: 30s

# Cache configuration
cache:
  directory: ~/.libreseed/npm-cache
  max_size: 10GB
  ttl: 7d

# Fallback configuration
fallback:
  # Use npm registry if LibreSeed lookup fails
  npm_registry: true
  # Timeout before falling back
  timeout: 30s

# Security
security:
  # Strict mode: reject unsigned packages
  strict: true
  # Trusted publishers (optional whitelist)
  trusted_publishers:
    - "ABC123..."
    - "DEF456..."
```

### Project Configuration

In `package.json`:

```json
{
  "libreseed": {
    "enabled": true,
    "packages": {
      "express": {
        "source": "libreseed",
        "publisher": "ABC123...",
        "version": "^4.18.0"
      },
      "lodash": {
        "source": "libreseed",
        "publisher": "DEF456..."
      }
    },
    "fallback": "npm"
  }
}
```

## Module Resolution Strategy

### Symlink Approach

LibreSeed packages are installed to cache and symlinked:

```
project/
├── node_modules/
│   ├── express → ~/.libreseed/npm-cache/express@4.18.2/
│   └── lodash → ~/.libreseed/npm-cache/lodash@4.17.21/
└── package.json

~/.libreseed/npm-cache/
├── express@4.18.2/
│   ├── package.json
│   ├── index.js
│   └── ...
└── lodash@4.17.21/
    ├── package.json
    ├── lodash.js
    └── ...
```

This approach:
- ✅ Works with Node.js module resolution
- ✅ Deduplicates packages across projects
- ✅ Supports nested dependencies
- ✅ Compatible with existing npm tooling

See [MODULE-RESOLUTION.md](MODULE-RESOLUTION.md) for detailed analysis.

## Compatibility

### Supported npm Features

- ✅ `npm install`
- ✅ `npm update`
- ✅ `npm list`
- ✅ `npm outdated`
- ✅ Semantic versioning
- ✅ Peer dependencies
- ✅ Dev dependencies
- ✅ Optional dependencies

### Limitations

- ⚠️ Bin scripts require manual linking
- ⚠️ Postinstall scripts run in cache directory
- ⚠️ Native modules require compilation after symlink

## Troubleshooting

### Package Not Found in LibreSeed

```
Error: Package 'mypackage@1.0.0' not found in LibreSeed DHT
```

**Solutions:**
- Verify package name and version
- Check publisher public key is correct
- Ensure DHT connectivity (check bootstrap nodes)
- Fall back to npm registry (set `fallback: "npm"`)

### Signature Verification Failed

```
Error: Signature verification failed for 'mypackage@1.0.0'
```

**Solutions:**
- Verify publisher public key in `package.json`
- Check manifest hasn't been tampered with
- Contact package publisher

### Symlink Creation Failed

```
Error: Failed to create symlink for 'mypackage'
```

**Solutions:**
- Check filesystem supports symlinks
- Verify permissions on `node_modules/` directory
- On Windows, ensure Developer Mode enabled

## Performance

### Cache Benefits

Cached packages are shared across projects:

```bash
# First project
cd project1
npm install  # Downloads express@4.18.2

# Second project
cd ../project2
npm install  # Reuses cached express@4.18.2 (instant)
```

### DHT Lookup Performance

- **First lookup**: 2-5 seconds (DHT query + download)
- **Cached lookup**: <100ms (local cache hit)
- **Concurrent lookups**: Parallelized for speed

## Security

### Signature Verification

Every package from LibreSeed is verified:

1. Fetch manifest from DHT
2. Extract signature and public key
3. Verify Ed25519 signature
4. Compare public key hash with configured publisher
5. Reject if verification fails

### Strict Mode

Enable strict mode for maximum security:

```yaml
security:
  strict: true
```

This:
- ✅ Rejects unsigned packages
- ✅ Requires exact publisher match
- ✅ Disallows fallback to npm
- ✅ Enforces checksum verification

## Contributing

See gateway-specific development guide:

```bash
git clone https://github.com/libreseed/libreseed-gateway-npm
cd libreseed-gateway-npm
npm install
npm test
```

## Related Documentation

- [Protocol Specification](../../spec/LIBRESEED-SPEC-v1.2.md) - Section 8: Gateway Integration
- [Feasibility Analysis](FEASIBILITY-ANALYSIS.md) - Technical deep dive
- [Integration Guide](INTEGRATION.md) - Detailed configuration
- [Module Resolution](MODULE-RESOLUTION.md) - How resolution works
