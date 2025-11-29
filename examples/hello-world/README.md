# Hello World Example

A simple LibreSeed package demonstrating basic package creation and distribution.

## Contents

- `main.go` - Simple Go program that prints a greeting
- `README.md` - This file

## Usage

```bash
go run main.go
```

## Building with LibreSeed Packager

```bash
# Generate a keypair (if you haven't already)
packager keygen -o mykey.private

# Create the package
packager create -key mykey.private -dir . -out ./dist -name hello-world -version 1.0.0
```

This will generate:
- `hello-world@1.0.0.tgz` - The package tarball
- `hello-world@1.0.0.minimal.json` - The signed manifest for distribution
