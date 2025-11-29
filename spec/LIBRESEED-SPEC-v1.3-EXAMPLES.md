# LIBRESEED Protocol Specification v1.3 - Examples

> Part of the [LIBRESEED Protocol Specification v1.3](./LIBRESEED-SPEC-v1.3-INDEX.md)

---

**Navigation:** [← Implementation Guide](./LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md) | [Index](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Glossary →](./LIBRESEED-SPEC-v1.3-GLOSSARY.md)

---

## 14. 📘 Examples

### 14.1 Publish Workflow

```bash
# 1. Generate keypair (first time only)
libreseed-packager keygen

# 2. Create package
cd mypackage/
libreseed-packager init

# 3. Build package
npm run build  # or your build process

# 4. Publish to LibreSeed
libreseed-packager publish \
    --name mypackage \
    --version 1.4.0 \
    --dist ./dist \
    --key ~/.libreseed/keys/packager.key

# Output:
# ✓ ContentHash computed from files
# ✓ Full manifest created and signed (contentHash signature)
# ✓ Tarball created: mypackage-1.4.0.tgz (includes full manifest)
# ✓ Minimal manifest created and signed (infohash signature)
# ✓ Published to DHT: sha256(libreseed:manifest:mypackage@1.4.0)
# ✓ Updated announce: sha256(libreseed:announce:<pubkey>)
# ✓ Updated Name Index: sha256(libreseed:name-index:mypackage)  [NEW]
# ✓ Seeding started
```

---

### 14.2 Seeder Deployment (Docker)

```bash
# 1. Create seeder config
cat > seeder.yaml <<EOF
trackedPublishers:
  - "ABC123..."  # Publisher public keys
trackedPackages:   # NEW in v1.3: Track by name
  - "mypackage"
  - "otherpackage"
maxDiskGB: 100
storagePath: "/data/libreseed"
EOF

# 2. Run seeder
docker run -d \
    --name libreseed-seeder \
    -v $(pwd)/seeder.yaml:/config/seeder.yaml \
    -v libreseed-data:/data/libreseed \
    -p 6881:6881 \
    libreseed/seeder:latest
```

---

### 14.3 User Installation (Direct CLI - Simplified with Name Index)

```bash
# NEW in v1.3: No publisher required!
libreseed-cli install mypackage@^1.4.0

# Output:
# ✓ Querying Name Index for 'mypackage'...
# ✓ Found 3 publishers
# ✓ Selected publisher: ABC123... (first-seen policy)
# ✓ Resolved: mypackage@1.4.2
# ✓ Downloading from 5 seeders...
# ✓ Verified signature
# ✓ Installed to ~/.libreseed/packages/abc123.../

# Alternative: Explicit packager (backwards compatible)
libreseed-cli install \
    --name mypackage \
    --version "^1.4.0" \
    --packager "ABC123..."
```

---

### 14.4 Query Name Index (NEW in v1.3)

```bash
# Query all publishers for a package
libreseed-cli query mypackage

# Output:
# Package: mypackage
# 
# Publisher 1:
#   Pubkey: ABC123...
#   Latest Version: 1.4.0
#   First Seen: 2024-11-20 14:30:00
#   Signature: Valid ✓
# 
# Publisher 2:
#   Pubkey: DEF456...
#   Latest Version: 1.3.5
#   First Seen: 2024-11-15 10:15:00
#   Signature: Valid ✓
# 
# Publisher 3:
#   Pubkey: GHI789...
#   Latest Version: 1.4.1
#   First Seen: 2024-11-22 09:00:00
#   Signature: Valid ✓
```

---

**Navigation:** [← Implementation Guide](./LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md) | [Index](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Glossary →](./LIBRESEED-SPEC-v1.3-GLOSSARY.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
