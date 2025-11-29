# LibreSeed Examples

This directory contains example packages demonstrating LibreSeed usage.

## Available Examples

### 1. hello-world
**Complexity:** Basic  
**Description:** A simple "Hello World" program demonstrating minimal package structure.

```bash
cd hello-world
packager create -key ../test.key -dir . -out ./dist -name hello-world -version 1.0.0
```

### 2. math-utils
**Complexity:** Intermediate  
**Description:** A collection of mathematical utility functions showing library-style packaging.

```bash
cd math-utils
packager create -key ../test.key -dir . -out ./dist -name math-utils -version 2.0.0
```

### 3. data-processor
**Complexity:** Advanced  
**Description:** A data processing tool with configuration files and multiple modules.

```bash
cd data-processor
packager create -key ../test.key -dir . -out ./dist -name data-processor -version 1.5.0
```

## Quick Start

### 1. Generate a Test Key

```bash
cd examples
../packager/build/packager keygen -o test.key
```

### 2. Create Your First Package

```bash
cd hello-world
../../packager/build/packager create \
  -key ../test.key \
  -dir . \
  -out ./dist \
  -name hello-world \
  -version 1.0.0
```

### 3. Verify the Package

```bash
../../packager/build/packager inspect dist/hello-world@1.0.0.minimal.json
```

### 4. Seed the Package

```bash
../../seeder/build/seeder add-package dist/hello-world@1.0.0.tgz
../../seeder/build/seeder start
```

## File Structure

Each example package contains:
- **Source code** - Go files or other source code
- **README.md** - Documentation for the example
- **dist/** - Generated packages (created after building):
  - `{name}@{version}.tgz` - Tarball with source code and manifest
  - `{name}@{version}.minimal.json` - Lightweight manifest for DHT
  - `{name}@{version}.torrent` - BitTorrent metainfo file

## Distribution Workflow

1. **Create** - Use Packager to create `.tgz` and `.minimal.json`
2. **Seed** - Use Seeder to make the package available on P2P network
3. **Distribute** - Share the `.minimal.json` file (small, signed manifest)
4. **Download** - Users fetch the actual `.tgz` from the P2P network using the infohash

## Testing End-to-End

```bash
# Generate key
packager keygen -o examples/test.key

# Create package
cd examples/hello-world
packager create -key ../test.key -dir . -out ./dist -name hello-world -version 1.0.0

# Inspect manifest
packager inspect dist/hello-world@1.0.0.minimal.json

# Add to seeder
cd ../../seeder
./build/seeder add-package ../examples/hello-world/dist/hello-world@1.0.0.tgz

# Start seeding
./build/seeder start
```

## Next Steps

After trying these examples, explore:
- Creating your own packages
- Setting up a gateway (npm, pip, cargo)
- Running multiple seeders for redundancy
- Building a package index/registry
