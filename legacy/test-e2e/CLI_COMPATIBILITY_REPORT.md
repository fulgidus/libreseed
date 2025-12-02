# LibreSeed CLI Command Compatibility Report

**Date:** 2025-11-29  
**Tested Components:** Packager v1.3, Seeder (latest build)  
**Test Suite:** End-to-End Package Flow Test  

---

## Executive Summary

✅ **Packager CLI:** Fully compatible  
⚠️ **Seeder CLI:** Flag naming mismatch detected

**Impact:** Low (functionality works, test script needs update)

---

## Packager Commands

### ✅ All Commands Compatible

| Command | Status | Flags | Notes |
|---------|--------|-------|-------|
| `packager generate-key` | ✅ PASS | `-o, --output` | Generates Ed25519 keypair |
| `packager build <dir>` | ✅ PASS | `-k, --key` | Creates .tgz + .minimal.json + .torrent |
| `packager inspect <file>` | ✅ PASS | None | Displays package metadata |

**No issues detected.** All commands work as expected.

---

## Seeder Commands

### ⚠️ Flag Naming Mismatch

#### Command: `seeder add-package`

**Test Script Expected:**
```bash
seeder add-package --tarball <file> --minimal <file>
```

**Actual Implementation:**
```bash
seeder add-package --package <file> --manifest <file>
```

**Detailed Comparison:**

| Flag | Test Script | Actual CLI | Alias | Status |
|------|-------------|------------|-------|--------|
| Package file | `--tarball` | `--package` or `-p` | ❌ No | ⚠️ **Mismatch** |
| Manifest file | `--minimal` | `--manifest` or `-m` | ❌ No | ⚠️ **Mismatch** |
| Timeout | `--timeout` | `--timeout` or `-t` | ✅ Yes | ✅ Compatible |
| Config | `--config` | `--config` | ✅ Yes | ✅ Compatible |

**Current Error When Using Test Script Flags:**
```
Error: unknown flag: --tarball
Usage:
  seeder add-package [flags]

Flags:
  -h, --help               help for add-package
  -m, --manifest string    Path to the minimal manifest JSON file (required)
  -p, --package string     Path to the .tgz package file (required)
  -t, --timeout duration   Timeout for adding package (default 1m0s)
```

---

## Impact Analysis

### Affected Components

1. ✅ **Packager** - No impact
2. ⚠️ **Test Suite** - `test-e2e/run-e2e-test.sh` uses incorrect flags
3. ℹ️ **Documentation** - May reference incorrect flag names
4. ℹ️ **User Scripts** - Community scripts may use expected flags

### Severity: **LOW**

**Reasoning:**
- Core functionality works perfectly
- Only flag *names* differ, not functionality
- Easy to fix (1-line change in test script)
- No runtime or logic errors

---

## Root Cause Analysis

### Possible Causes

1. **Design Evolution** - Flags renamed during implementation
2. **Specification Ambiguity** - Spec didn't prescribe exact flag names
3. **Test Script Pre-dates Implementation** - Script written before final CLI

### Most Likely Cause

Test script was created based on expected/planned CLI, but implementation used different (arguably better) flag names:
- `--package` is more generic than `--tarball` (format-agnostic)
- `--manifest` is more precise than `--minimal` (describes purpose, not size)

---

## Recommendations

### Option 1: Update Test Script (Recommended) ✅

**Effort:** 5 minutes  
**Impact:** Low  
**Pros:** 
- Aligns test with actual implementation
- No code changes needed in seeder
- Immediate fix

**Changes Required:**
```diff
# test-e2e/run-e2e-test.sh:87-89
  $SEEDER add-package \
-     --tarball "${PKG_NAME}@${PKG_VERSION}.tgz" \
-     --minimal "${PKG_NAME}@${PKG_VERSION}.minimal.json"
+     --package "${PKG_NAME}@${PKG_VERSION}.tgz" \
+     --manifest "${PKG_NAME}@${PKG_VERSION}.minimal.json"
```

---

### Option 2: Add Flag Aliases (Alternative)

**Effort:** 2-4 hours  
**Impact:** Medium  
**Pros:**
- Backward compatibility for users
- Test script works as-is
- More user-friendly (accepts both styles)

**Changes Required:**
```go
// seeder/internal/cli/add_package.go
cmd.Flags().StringP("package", "p", "", "Path to the .tgz package file (required)")
cmd.Flags().StringP("manifest", "m", "", "Path to the minimal manifest JSON file (required)")

// Add aliases
cmd.Flags().StringP("tarball", "", "", "Alias for --package (deprecated)")
cmd.Flags().StringP("minimal", "", "", "Alias for --manifest (deprecated)")

// In command execution, check both:
packagePath := cmd.Flags().GetString("package")
if packagePath == "" {
    packagePath = cmd.Flags().GetString("tarball")
}
```

**Pros:**
- Supports both old and new flag names
- Smooth migration path

**Cons:**
- More code complexity
- May confuse users with multiple options
- Requires deprecation notice

---

### Option 3: Do Nothing (Not Recommended) ❌

**Effort:** 0 hours  
**Impact:** Low (test fails, but manual validation works)  

**Cons:**
- Test suite remains broken
- Misleading for new developers
- Documentation may be inaccurate

---

## Corrected Usage Examples

### Correct Command Syntax

```bash
# Add a package to seeder
seeder add-package \
  --package "hello-test@1.0.0.tgz" \
  --manifest "hello-test@1.0.0.minimal.json"

# With configuration file
seeder add-package \
  --config seeder.yaml \
  --package "hello-test@1.0.0.tgz" \
  --manifest "hello-test@1.0.0.minimal.json"

# With timeout
seeder add-package \
  --package "hello-test@1.0.0.tgz" \
  --manifest "hello-test@1.0.0.minimal.json" \
  --timeout 2m
```

### Short Flags

```bash
# Using short flag aliases
seeder add-package -p "hello-test@1.0.0.tgz" -m "hello-test@1.0.0.minimal.json"
```

---

## Testing Verification

### Verified Working Commands ✅

```bash
# ✅ This works
cd test-e2e
../seeder/build/seeder add-package \
  --config seeder.yaml \
  --package "hello-test@1.0.0.tgz" \
  --manifest "hello-test@1.0.0.minimal.json"

# Result:
✓ Package validation successful
✓ Package added to seeder
✓ DHT announcement triggered
```

### Verified Failing Commands ❌

```bash
# ❌ This fails
seeder add-package \
  --tarball "hello-test@1.0.0.tgz" \
  --minimal "hello-test@1.0.0.minimal.json"

# Error:
Error: unknown flag: --tarball
```

---

## Documentation Updates Needed

### Files to Update

1. **test-e2e/run-e2e-test.sh** (lines 87-89)
   - Change `--tarball` → `--package`
   - Change `--minimal` → `--manifest`

2. **test-e2e/E2E-TEST-INSTRUCTIONS.md** (if exists)
   - Update command examples
   - Add flag reference table

3. **seeder/README.md**
   - Verify flag documentation accuracy
   - Add examples using correct flags

4. **Root README.md**
   - Check for outdated examples
   - Update quick-start guides

---

## Implementation Plan

### Phase 1: Immediate Fix (Day 1)

**Goal:** Make test suite pass

- [ ] Update `test-e2e/run-e2e-test.sh` with correct flags
- [ ] Run full test suite to verify fix
- [ ] Document changes in commit message

**Estimated Time:** 15 minutes

---

### Phase 2: Documentation (Day 1-2)

**Goal:** Ensure all docs use correct syntax

- [ ] Audit all README files
- [ ] Search for `--tarball` and `--minimal` references
- [ ] Update examples and tutorials
- [ ] Add CLI reference documentation

**Estimated Time:** 2 hours

---

### Phase 3: Communication (Optional)

**Goal:** Inform users of correct syntax

- [ ] Add note to release notes
- [ ] Update CHANGELOG
- [ ] Create migration guide (if aliases added)

**Estimated Time:** 1 hour

---

## Acceptance Criteria

✅ Test suite passes without manual flag correction  
✅ All documentation uses correct flag names  
✅ No breaking changes introduced  
✅ Users can successfully run examples from documentation  

---

## Conclusion

**The CLI flag naming mismatch is a minor documentation/test issue with ZERO functional impact.**

### Summary

- ✅ Seeder functionality: **Perfect**
- ⚠️ Test script: **Needs 2-line update**
- ℹ️ Documentation: **May need review**

### Recommended Action

**Update test script immediately** (15 minutes) to unblock automated testing.

### Long-term

Consider adding flag aliases for better UX, but not critical for v1.3 release.

---

**Report By:** White Box Testing Agent  
**Priority:** P2 (Non-blocking)  
**Estimated Fix Time:** 15 minutes  
**Status:** Ready for Developer Action
