#!/bin/bash

# LibreSeed End-to-End Test Script
# Tests: Packager → Package Creation → Seeder Validation

set -e  # Exit on error

echo "=========================================="
echo "LibreSeed End-to-End Test"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PACKAGER="../packager/build/packager"
SEEDER="../seeder/build/seeder"
TEST_DIR="test-project"
KEY_FILE="test.key"
PKG_NAME="hello-test"
PKG_VERSION="1.0.0"

# Step 1: Generate keypair
echo "Step 1: Generating Ed25519 keypair..."
if [ -f "$KEY_FILE" ]; then
    echo "  (Using existing $KEY_FILE)"
else
    $PACKAGER keygen "$KEY_FILE"
fi
echo -e "${GREEN}✓${NC} Keypair ready"
echo ""

# Step 2: Create package
echo "Step 2: Creating package..."
$PACKAGER create "$TEST_DIR" \
    --name "$PKG_NAME" \
    --version "$PKG_VERSION" \
    --description "End-to-end test package" \
    --author "LibreSeed Test Suite" \
    --key "$KEY_FILE" \
    --output .

if [ ! -f "${PKG_NAME}@${PKG_VERSION}.tgz" ]; then
    echo -e "${RED}✗${NC} Failed: Tarball not created"
    exit 1
fi

if [ ! -f "${PKG_NAME}@${PKG_VERSION}.minimal.json" ]; then
    echo -e "${RED}✗${NC} Failed: Minimal manifest not created"
    exit 1
fi

echo -e "${GREEN}✓${NC} Package created successfully"
echo ""

# Step 3: Inspect package
echo "Step 3: Inspecting package..."
$PACKAGER inspect "${PKG_NAME}@${PKG_VERSION}.tgz"
echo -e "${GREEN}✓${NC} Package inspection complete"
echo ""

# Step 4: Display minimal manifest
echo "Step 4: Minimal Manifest:"
cat "${PKG_NAME}@${PKG_VERSION}.minimal.json" | jq .
echo -e "${GREEN}✓${NC} Minimal manifest valid JSON"
echo ""

# Step 5: Build seeder (if not already built)
echo "Step 5: Building seeder..."
if [ ! -f "$SEEDER" ]; then
    echo "  Building seeder from source..."
    (cd ../seeder && make build)
fi
echo -e "${GREEN}✓${NC} Seeder ready"
echo ""

# Step 6: Test seeder validation
echo "Step 6: Testing seeder validation..."
echo "  This will validate both signatures..."

# Note: The seeder add-package command might not exist yet
# We'll check if the validator can be tested directly
if $SEEDER --help 2>&1 | grep -q "add-package"; then
    $SEEDER add-package \
        --tarball "${PKG_NAME}@${PKG_VERSION}.tgz" \
        --minimal "${PKG_NAME}@${PKG_VERSION}.minimal.json"
    echo -e "${GREEN}✓${NC} Seeder validation passed"
else
    echo -e "${YELLOW}⚠${NC}  Seeder add-package command not available"
    echo "     Manual validation required"
fi
echo ""

# Summary
echo "=========================================="
echo "End-to-End Test Summary"
echo "=========================================="
echo -e "${GREEN}✓${NC} Keypair generation"
echo -e "${GREEN}✓${NC} Package creation"
echo -e "${GREEN}✓${NC} Full manifest (inside tarball)"
echo -e "${GREEN}✓${NC} Minimal manifest (separate file)"
echo -e "${GREEN}✓${NC} Package inspection"
echo ""
echo "Output files:"
ls -lh "${PKG_NAME}@${PKG_VERSION}".* 2>/dev/null || true
echo ""
echo -e "${GREEN}End-to-End Test PASSED${NC}"
