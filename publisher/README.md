# LibreSeed Publisher Documentation

The **Publisher** is the tool used to publish packages to the LibreSeed network.

## Overview

Publishers create signed manifests and announce them to the DHT network. The publishing workflow:

1. Generate Ed25519 keypair (one-time)
2. Create package manifest
3. Sign manifest with private key
4. Announce manifest to DHT
5. (Optional) Upload files to seeders

## Quick Start

### Generate Keypair

```bash
libreseed keygen --output publisher-key.pem
```

This creates an Ed25519 keypair. **Keep the private key secure!**

### Publish a Package

```bash
libreseed publish \
  --name mypackage \
  --version 1.0.0 \
  --key publisher-key.pem \
  --tarball mypackage-1.0.0.tgz \
  --seeder https://seed1.example.com
```

### Verify Publication

```bash
libreseed verify \
  --name mypackage \
  --version 1.0.0 \
  --pubkey publisher-key-public.pem
```

## Documentation

- **[PUBLISHING.md](PUBLISHING.md)** - Complete publishing workflow guide
- **[MANIFEST-FORMAT.md](MANIFEST-FORMAT.md)** - Manifest structure and fields
- **[KEYGEN.md](KEYGEN.md)** - Keypair generation and management

## Publishing Workflow

### 1. Prepare Package

Create your package tarball:

```bash
# For npm packages
npm pack

# For Python packages
python setup.py sdist

# For Go modules
# (LibreSeed stores module source, not binaries)
```

### 2. Create Manifest

Create a manifest file `mypackage-1.0.0.manifest.json`:

```json
{
  "name": "mypackage",
  "version": "1.0.0",
  "description": "My awesome package",
  "publisher": "base64-encoded-public-key",
  "files": [
    {
      "name": "mypackage-1.0.0.tgz",
      "checksum": "sha256:abc123...",
      "size": 12345,
      "url": "https://seed1.example.com/files/abc123..."
    }
  ],
  "dependencies": {
    "otherpkg": "^2.0.0"
  },
  "timestamp": 1680000000
}
```

### 3. Sign Manifest

```bash
libreseed sign \
  --manifest mypackage-1.0.0.manifest.json \
  --key publisher-key.pem \
  --output mypackage-1.0.0.manifest.signed.json
```

### 4. Announce to DHT

```bash
libreseed announce \
  --manifest mypackage-1.0.0.manifest.signed.json
```

This publishes to DHT key:
```
sha256("libreseed:manifest:mypackage@1.0.0")
```

### 5. Upload Files to Seeders

```bash
libreseed upload \
  --manifest mypackage-1.0.0.manifest.signed.json \
  --seeder https://seed1.example.com \
  --key publisher-key.pem
```

## Publisher Identity

Your publisher identity is derived from your public key:

```
publisher_id = base64(ed25519_pubkey)
```

Users can verify packages from your publisher ID:

```bash
libreseed verify --name mypackage --version 1.0.0 --publisher YOUR_ID
```

## Security Best Practices

### Private Key Management

- **Never commit private keys to version control**
- Store private keys encrypted at rest
- Use hardware security modules (HSM) for production
- Rotate keys if compromised (requires new publisher identity)

### Signing Workflow

- Always verify manifest content before signing
- Use canonical JSON encoding for consistent signatures
- Sign on trusted, air-gapped machines for critical packages
- Keep audit logs of all signatures

### Key Rotation

If your key is compromised:

1. Generate new keypair
2. Publish packages under new identity
3. Communicate new identity to users
4. **Note**: Old packages remain under old identity (no revocation mechanism)

## Automation

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Publish to LibreSeed

on:
  release:
    types: [published]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install LibreSeed CLI
        run: |
          wget https://github.com/libreseed/libreseed/releases/download/v1.0.0/libreseed-linux-amd64
          chmod +x libreseed-linux-amd64
          sudo mv libreseed-linux-amd64 /usr/local/bin/libreseed
      
      - name: Build package
        run: npm pack
      
      - name: Publish to LibreSeed
        env:
          LIBRESEED_KEY: ${{ secrets.LIBRESEED_PRIVATE_KEY }}
        run: |
          echo "$LIBRESEED_KEY" > key.pem
          libreseed publish \
            --name ${{ github.event.repository.name }} \
            --version ${{ github.event.release.tag_name }} \
            --key key.pem \
            --tarball *.tgz \
            --seeder https://seed1.example.com
          rm key.pem
```

## Troubleshooting

### "Signature verification failed"

- Ensure manifest hasn't been modified after signing
- Verify you're using correct private key
- Check timestamp is valid (not too far in past/future)

### "DHT announce failed"

- Check network connectivity
- Verify DHT bootstrap nodes are reachable
- Try announcing to multiple DHT bootstrap nodes

### "Seeder rejected upload"

- Verify seeder is configured to accept uploads
- Check authentication credentials
- Ensure manifest signature is valid
- Verify file checksums match manifest

## Related Documentation

- [Protocol Specification](../spec/LIBRESEED-SPEC-v1.2.md) - Section 6: Publisher Workflow
- [Architecture Review](../docs/architecture/ARCHITECTURE-REVIEW.md) - Publisher component
- [Test Strategy](../docs/testing/LIBRESEED_TEST_STRATEGY.md) - Publisher testing
