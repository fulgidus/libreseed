# LibreSeed Packager

**LibreSeed Packager** creates cryptographically signed packages for decentralized distribution via BitTorrent and DHT.

## Features

- **Ed25519 Cryptographic Signing** – All packages are signed with your private key
- **Two-Manifest System** – Full manifest in the tarball, minimal manifest for DHT
- **Content Integrity** – SHA256 hashes for every file and the package itself
- **Simple CLI** – Create, inspect, and sign packages with ease
- **Decentralized** – No central authority needed for distribution

## Installation

### Build from Source

```bash
# Clone the repository
cd /path/to/libreseed/packager

# Build the packager
make build

# (Optional) Install to system
make install
```

The binary will be available at `build/packager` (or `/usr/local/bin/packager` if installed).

## Quick Start

### 1. Generate a Keypair

First, generate an Ed25519 keypair for signing your packages:

```bash
packager keygen private.key
```

**Output:**
```
Generating Ed25519 keypair...
✓ Keypair generated successfully!
  Private key saved to: private.key
  Public key: ed25519:AbCd1234...

⚠️  Keep your private key secure and never share it!
```

Save the public key – you'll need it for seeder configuration.

### 2. Create a Package

```bash
packager create ./my-package \
  --name hello-world \
  --version 1.0.0 \
  --description "A sample package" \
  --author "Your Name" \
  --key private.key \
  --output ./dist
```

**Output:**
```
Building package hello-world@1.0.0 from ./my-package...
✓ Package created successfully!
  Tarball:          dist/hello-world@1.0.0.tgz
  Minimal manifest: dist/hello-world@1.0.0.minimal.json
  Torrent file:     dist/hello-world@1.0.0.torrent
```

### 3. Inspect a Package

```bash
packager inspect dist/hello-world@1.0.0.tgz
```

**Output:**
```
Inspecting package: dist/hello-world@1.0.0.tgz

Package Information:
  Name:        hello-world
  Version:     1.0.0
  Description: A sample package
  Author:      Your Name

Cryptographic Information:
  ContentHash: sha256:abc123...
  Public Key:  ed25519:XyZ789...
  Signature:   ed25519:Sig456...

Files (3 total):
  main.go
    sha256:file1hash...
  README.md
    sha256:file2hash...
  lib/util.go
    sha256:file3hash...
```

## Usage

### Commands

#### `packager create <directory>`

Create a signed package from a directory.

**Flags:**
- `-n, --name` (required) – Package name
- `-v, --version` (required) – Package version (semver recommended)
- `-k, --key` (required) – Private key file or hex string
- `-d, --description` – Package description
- `-a, --author` – Package author
- `-o, --output` – Output directory (default: current directory)
- `--include-hidden` – Include hidden files (starting with `.`)

**Example:**
```bash
packager create ./my-app \
  --name my-app \
  --version 2.1.0 \
  --key ./keys/private.key \
  --description "My awesome application" \
  --author "Alice <alice@example.com>" \
  --output ./releases
```

#### `packager keygen <output-file>`

Generate a new Ed25519 keypair.

**Example:**
```bash
packager keygen ~/.libreseed/private.key
```

#### `packager inspect <package.tgz>`

Inspect a package and display its manifest.

**Example:**
```bash
packager inspect my-app@2.1.0.tgz
```

## Output Format

When you create a package, three files are generated:

### 1. **Tarball** (`{name}@{version}.tgz`)

A gzipped tarball containing:
- `manifest.json` – Full manifest with all file hashes and signatures
- All package files in their original structure

### 2. **Minimal Manifest** (`{name}@{version}.minimal.json`)

A lightweight JSON file for DHT announcements containing:
- Package name and version
- Public key
- Infohash (SHA256 of the tarball)
- Infohash signature

### 3. **Torrent File** (`{name}@{version}.torrent`)

A standard BitTorrent `.torrent` file containing:
- Metainfo structure with piece hashes (256KB pieces)
- SHA256-based piece verification
- Single-file torrent pointing to the `.tgz`

**Note:** The seeder generates its own torrent internally when adding packages, so the `.torrent` file is optional for seeding but useful for verification, distribution, or use with standard BitTorrent clients.

## Manifest Structure

### Full Manifest (inside .tgz)

```json
{
  "name": "hello-world",
  "version": "1.0.0",
  "description": "A sample package",
  "author": "Your Name",
  "files": {
    "main.go": "sha256:abc123...",
    "README.md": "sha256:def456..."
  },
  "contentHash": "sha256:xyz789...",
  "pubKey": "ed25519:AbCd1234...",
  "signature": "ed25519:Sig456..."
}
```

### Minimal Manifest (for DHT)

```json
{
  "name": "hello-world",
  "version": "1.0.0",
  "pubKey": "ed25519:AbCd1234...",
  "infohash": "sha256:tgzhash...",
  "infohashSignature": "ed25519:TgzSig..."
}
```

## Cryptographic Details

### Signing Process

1. **File Hashing** – Each file is hashed with SHA256
2. **ContentHash Calculation** – All file hashes are sorted, concatenated, and hashed again
3. **ContentHash Signing** – The contentHash is signed with Ed25519 private key
4. **Tarball Creation** – Files + full manifest packed into `.tgz`
5. **Infohash Calculation** – The entire tarball is hashed with SHA256
6. **Infohash Signing** – The infohash is signed with Ed25519 private key
7. **Minimal Manifest** – Written with infohash + signature
8. **Torrent Generation** – BitTorrent `.torrent` file created from the `.tgz`

### Hash and Signature Formats

- **SHA256 hashes:** `sha256:<64-char-hex>`
- **Ed25519 public keys:** `ed25519:<base64>`
- **Ed25519 signatures:** `ed25519:<base64>`

## Integration with LibreSeed Seeder

After creating a package:

1. **Add to Seeder:**
   ```bash
   # Using seeder CLI (recommended)
   seeder add-package \
     --config seeder.yaml \
     --package dist/hello-world@1.0.0.tgz \
     --manifest dist/hello-world@1.0.0.minimal.json
   
   # Start seeding
   seeder start --config seeder.yaml
   ```

2. **Seeder validates signatures** using the public key
3. **Seeder generates torrent** and starts seeding the `.tgz`
4. **Seeder announces to DHT** using the computed InfoHash
5. **Clients download** via BitTorrent and verify using full manifest

**Note:** The `.torrent` file generated by packager can be distributed separately for use with standard BitTorrent clients, but is not required by the seeder.

## Security Notes

- **Never share your private key** – Treat it like a password
- **Backup your keys** – Loss of private key means you can't sign new versions
- **Rotate keys carefully** – Clients will need the new public key
- **Verify signatures** – Always check signatures before distributing

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Clean

```bash
make clean
```

## Architecture

For detailed architectural decisions, see [DESIGN_DECISIONS.md](../DESIGN_DECISIONS.md) in the project root.

## License

[Specify license here]

## Contributing

[Specify contribution guidelines here]
