# LibreSeed

**Decentralized Package Distribution via DHT**

LibreSeed is a protocol and toolset for distributing software packages over a peer-to-peer DHT network, eliminating the need for centralized registries.

## 🚀 Quick Start

### For Users

```bash
# Install LibreSeed CLI
curl -sSL https://get.libreseed.com | sh

# Install a package via gateway
cd your-project
npm install  # Automatically uses LibreSeed for configured packages
```

### For Publishers

```bash
# Generate keypair
libreseed keygen --output publisher-key.pem

# Publish a package
libreseed publish \
  --name mypackage \
  --version 1.0.0 \
  --key publisher-key.pem \
  --tarball mypackage-1.0.0.tgz
```

### For Seeder Operators

```bash
# Run a seeder
docker run -d \
  --name libreseed-seeder \
  -v /var/lib/libreseed:/var/lib/libreseed \
  -p 6881:6881/udp \
  -p 8080:8080 \
  libreseed/seeder:latest
```

## 🎯 Key Features

### Decentralized Architecture
- **No Central Registry**: Packages distributed via BitTorrent mainline DHT
- **Pure P2P**: Direct peer-to-peer discovery and download
- **Censorship Resistant**: No single point of failure or control

### Security First
- **Cryptographic Signing**: Ed25519 signatures on all manifests
- **Publisher Identity**: Packages tied to publisher public keys
- **Integrity Verification**: Checksums validated at every step
- **No Trust Required**: Verify signatures, don't trust intermediaries

### Multi-Gateway Support
- **NPM Gateway**: Seamless integration with existing npm workflows
- **Extensible**: Design supports pip, cargo, gem, and other package managers
- **Backward Compatible**: Falls back to traditional registries when needed

### Efficient Distribution
- **DHT-Based Discovery**: Fast, distributed package lookup
- **BitTorrent Protocol**: Efficient file distribution with deduplication
- **Seeder Network**: Volunteer nodes provide availability and redundancy
- **Local Caching**: Shared cache across projects

## 🔐 Security Model

### Publisher Identity
```
publisher_id = sha256(ed25519_pubkey)[:20]
```

### Manifest Signing
```
1. Create canonical JSON manifest
2. Sign with Ed25519 private key
3. Attach signature and public key
4. Publish to DHT
```

### Verification Chain
```
Client → DHT → Manifest → Signature → Public Key → Publisher ID
   ✓      ✓       ✓          ✓            ✓            ✓
```

## 🛠️ Implementation Status

### ✅ Completed
- Protocol specification v1.2
- Architecture design and analysis
- Comprehensive test strategy
- Documentation structure

### 🔄 In Progress
- Seeder implementation (Go)
- Publisher CLI (Go)
- NPM Gateway (Node.js)
- DHT client library

### ⏳ Planned
- Python gateway (pip)
- Rust gateway (cargo)
- Ruby gateway (gem)
- Web UI for browsing packages

## 🤝 Contributing

LibreSeed is open-source and welcomes contributions.

### Development Setup

TBD

## 📜 License

MIT License - see [LICENSE](LICENSE) for details.

## 🔗 Links

- **Website**: https://libreseed.org (TODO)
- **GitHub**: https://github.com/libreseed/libreseed
- **Specification**: [spec/LIBRESEED-SPEC-v1.2.md](spec/LIBRESEED-SPEC-v1.2.md)
- **Discussion**: GitHub Discussions

## 💬 Community

TBD

---

**LibreSeed** - Decentralizing package distribution, one package at a time.
