# End-to-End Test Instructions

This document provides step-by-step instructions to test the complete LibreSeed pipeline: package creation → validation → seeding.

## Prerequisites

- Go 1.21+ installed
- `jq` for JSON inspection (optional but recommended)
- Terminal with bash

## Quick Start

```bash
cd /home/fulgidus/Documents/libreseed/test-e2e
chmod +x run-e2e-test.sh
./run-e2e-test.sh
```

## Manual Step-by-Step Testing

If you prefer to run each step manually:

### Step 1: Generate Keypair

```bash
cd /home/fulgidus/Documents/libreseed/test-e2e
../packager/build/packager keygen test.key
```

**Expected Output:**
```
Generating Ed25519 keypair...
✓ Keypair generated successfully!
  Private key saved to: test.key
  Public key: ed25519:<64-char-hex>

⚠️  Keep your private key secure and never share it!
```

**Verify:**
- File `test.key` created
- Public key displayed in format `ed25519:<hex>`

---

### Step 2: Create Package

```bash
../packager/build/packager create test-project \
  --name hello-test \
  --version 1.0.0 \
  --description "End-to-end test package" \
  --author "LibreSeed Test Suite" \
  --key test.key \
  --output .
```

**Expected Output:**
```
Building package hello-test@1.0.0 from test-project...
✓ Package created successfully!
  Tarball:         hello-test@1.0.0.tgz
  Minimal manifest: hello-test@1.0.0.minimal.json
```

**Verify:**
```bash
ls -lh hello-test@1.0.0.*
```

Should show:
- `hello-test@1.0.0.tgz` (tarball with files + full manifest)
- `hello-test@1.0.0.minimal.json` (DHT manifest)

---

### Step 3: Inspect Full Manifest

```bash
../packager/build/packager inspect hello-test@1.0.0.tgz
```

**Expected Output:**
```
Inspecting package: hello-test@1.0.0.tgz

Package Information:
  Name:        hello-test
  Version:     1.0.0
  Description: End-to-end test package
  Author:      LibreSeed Test Suite

Cryptographic Information:
  ContentHash: sha256:<64-char-hex>
  Public Key:  ed25519:<64-char-hex>
  Signature:   <128-char-hex>

Files (2 total):
  index.js
    sha256:<64-char-hex>
  README.md
    sha256:<64-char-hex>
```

**What to Verify:**
- ✅ ContentHash is a SHA256 hash (64 hex chars)
- ✅ Public key matches the one from keygen
- ✅ Signature is 128 hex chars (Ed25519 signature)
- ✅ All files from test-project are listed
- ✅ Each file has a SHA256 hash

---

### Step 4: Inspect Minimal Manifest

```bash
cat hello-test@1.0.0.minimal.json | jq .
```

**Expected Output:**
```json
{
  "name": "hello-test",
  "version": "1.0.0",
  "infohash": "sha256:<64-char-hex>",
  "pubKey": "ed25519:<64-char-hex>",
  "signature": "<128-char-hex>"
}
```

**What to Verify:**
- ✅ Infohash is a SHA256 hash (64 hex chars)
- ✅ Public key matches full manifest
- ✅ Signature is 128 hex chars (different from full manifest signature)
- ✅ Name and version match

**Critical Check:**
```bash
# Public keys must match
FULL_PUBKEY=$(tar -xzOf hello-test@1.0.0.tgz manifest.json | jq -r .pubKey)
MINIMAL_PUBKEY=$(jq -r .pubKey hello-test@1.0.0.minimal.json)

if [ "$FULL_PUBKEY" = "$MINIMAL_PUBKEY" ]; then
  echo "✓ Public keys match"
else
  echo "✗ ERROR: Public keys don't match!"
fi
```

---

### Step 5: Verify Dual Signature Model

Extract and compare both manifests:

```bash
# Extract full manifest
tar -xzOf hello-test@1.0.0.tgz manifest.json > full-manifest.json

# Compare
echo "=== FULL MANIFEST (signs contentHash) ==="
jq '{contentHash, pubKey, signature}' full-manifest.json

echo ""
echo "=== MINIMAL MANIFEST (signs infohash) ==="
jq '{infohash, pubKey, signature}' hello-test@1.0.0.minimal.json
```

**What to Verify:**
- ✅ Full manifest has `contentHash` field
- ✅ Minimal manifest has `infohash` field
- ✅ Both have same `pubKey`
- ✅ Both have different `signature` values

**Why signatures differ:**
- Full manifest signs: SHA256(sorted file hashes) = contentHash
- Minimal manifest signs: SHA256(entire .tgz file) = infohash
- Same keypair, different data → different signatures ✓

---

### Step 6: Test Seeder (Optional)

If the seeder is built and has CLI commands:

```bash
cd ../seeder
make build
./build/seeder add-package \
  --package ../test-e2e/hello-test@1.0.0.tgz \
  --manifest ../test-e2e/hello-test@1.0.0.minimal.json
```

**Expected Behavior:**
1. Seeder extracts full manifest from tarball
2. Validates `contentHash` signature against full manifest
3. Computes infohash of tarball
4. Validates `infohash` signature against minimal manifest
5. Checks both manifests have matching public keys
6. Accepts package if all validations pass

---

## Success Criteria

### ✅ Package Creation
- [x] Keypair generated
- [x] Both output files created
- [x] No errors during build

### ✅ Full Manifest Validation
- [x] Contains all files with SHA256 hashes
- [x] Has `contentHash` field
- [x] Has valid Ed25519 signature
- [x] Public key in correct format

### ✅ Minimal Manifest Validation
- [x] Contains `infohash` field
- [x] Has valid Ed25519 signature
- [x] Public key matches full manifest
- [x] Signature differs from full manifest

### ✅ Dual Signature Architecture
- [x] Full manifest signs `contentHash` (file content integrity)
- [x] Minimal manifest signs `infohash` (tarball integrity)
- [x] Both use same keypair
- [x] Signatures are independent and different

---

## Troubleshooting

### Error: "command not found"

**Problem:** Packager binary not built

**Solution:**
```bash
cd /home/fulgidus/Documents/libreseed/packager
make build
```

### Error: Invalid signature format

**Problem:** Key file corrupted or wrong format

**Solution:** Regenerate keypair:
```bash
rm test.key
../packager/build/packager keygen test.key
```

### Public keys don't match

**Problem:** Critical error - manifests signed with different keys

**Solution:** This should never happen. Check packager implementation.

### Missing files in package

**Problem:** Files excluded during packaging

**Solution:** Check if files start with `.` (hidden). Use `--include-hidden` flag if needed.

---

## Next Steps After Testing

1. **If all tests pass:**
   - Document test results
   - Proceed to integration testing with live seeder
   - Test DHT announcement workflow

2. **If tests fail:**
   - Check error messages
   - Verify Go version (1.21+)
   - Review packager logs
   - Inspect generated files manually

---

## Clean Up

```bash
# Remove test artifacts
rm -f test.key
rm -f hello-test@1.0.0.*
rm -f full-manifest.json
```

---

## Files Generated

| File | Description | Contains |
|------|-------------|----------|
| `test.key` | Private key | Ed25519 private key (hex) |
| `hello-test@1.0.0.tgz` | Package tarball | Files + full manifest |
| `hello-test@1.0.0.minimal.json` | DHT manifest | Minimal metadata + infohash sig |
| `full-manifest.json` | Extracted manifest | For inspection only |

---

## Architecture Verified

```
┌─────────────────────────────────────────────────────────┐
│                    PACKAGER                              │
│                                                          │
│  1. Hash all files → contentHash                        │
│  2. Sign contentHash with private key                   │
│  3. Create tarball with files + full manifest           │
│  4. Hash tarball → infohash                             │
│  5. Sign infohash with same private key                 │
│  6. Output minimal manifest separately                  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
         ┌─────────────────────────────────┐
         │   TWO OUTPUT FILES               │
         ├─────────────────────────────────┤
         │ 1. package.tgz                  │
         │    - Files                       │
         │    - manifest.json               │
         │      • contentHash + signature   │
         │                                  │
         │ 2. package.minimal.json          │
         │    • infohash + signature        │
         └─────────────────────────────────┘
                           │
                           ▼
         ┌─────────────────────────────────┐
         │          SEEDER                  │
         │                                  │
         │  1. Validate contentHash sig     │
         │  2. Validate infohash sig        │
         │  3. Verify pubKey match          │
         │  4. Accept if all valid          │
         └─────────────────────────────────┘
```

This test verifies the complete dual-manifest architecture is working correctly.
