# LibreSeed Testing & Validation Plan

**Status:** Implementation Complete, Testing In Progress  
**Last Updated:** 2025-01-29

---

## ✅ Completed Work

### File Watcher Implementation
- ✅ Complete file watcher with fsnotify
- ✅ Automatic package detection and seeding
- ✅ File management (seeded/invalid directories)
- ✅ Debouncing (2-second delay)
- ✅ Graceful shutdown integration
- ✅ Configuration via `manifest.watch_dir`

**Commit:** `bf11ad46` - "feat(seeder): Add automatic file watcher for package directory"

---

## 🧪 Available Test Scripts

### 1. File Watcher Integration Test
**Location:** `seeder/test-watcher.sh`  
**Status:** ✅ Script created, ready to run  
**Purpose:** Validates automatic package detection and processing

**Tests Covered:**
1. ✅ Seeder startup with watcher enabled
2. ✅ Automatic package detection
3. ✅ File movement (seeded/invalid directories)
4. ✅ Invalid package handling
5. ✅ Multiple file processing
6. ✅ Seeder status verification
7. ✅ Graceful shutdown

**Run:**
```bash
cd seeder
chmod +x test-watcher.sh
./test-watcher.sh
```

**Expected Results:**
- All 7 test scenarios pass
- Files automatically moved to correct directories
- Clean startup and shutdown logs
- No crashes or errors

---

### 2. End-to-End Package Flow Test
**Location:** `test-e2e/run-e2e-test.sh`  
**Status:** ✅ Existing script, ready to run  
**Purpose:** Validates complete packager → seeder workflow

**Tests Covered:**
1. ✅ Keypair generation
2. ✅ Package creation (dual-manifest)
3. ✅ Package inspection
4. ✅ Minimal manifest validation
5. ✅ Seeder validation (if available)

**Run:**
```bash
cd test-e2e
chmod +x run-e2e-test.sh
./run-e2e-test.sh
```

**Expected Results:**
- Package created with both manifests
- Both signatures verify independently
- All validation steps pass
- Clean exit with summary

---

### 3. Manual File Watcher Test
**Location:** `seeder/FILE_WATCHER_TEST.md`  
**Status:** ✅ Comprehensive testing guide  
**Purpose:** Step-by-step manual validation

**Test Scenarios:**
1. Basic file watching
2. Invalid package handling
3. Multiple files
4. Duplicate detection
5. Large file debouncing
6. Graceful shutdown
7. Disabled watcher

**Run:**
Follow step-by-step instructions in `FILE_WATCHER_TEST.md`

---

## 🎯 Priority Testing Queue

### Priority 1: File Watcher Automated Test (IMMEDIATE)
**Action:**
```bash
cd seeder
chmod +x test-watcher.sh
./test-watcher.sh
```

**Success Criteria:**
- All 7 automated tests pass
- No errors in logs
- Files moved to correct directories
- Clean shutdown

**Time Estimate:** 2-3 minutes

---

### Priority 2: End-to-End Package Flow (HIGH)
**Action:**
```bash
cd test-e2e
chmod +x run-e2e-test.sh
./run-e2e-test.sh
```

**Success Criteria:**
- Package created successfully
- Both manifests generated
- Signatures verify
- Inspection shows correct data

**Time Estimate:** 1-2 minutes

---

### Priority 3: Manual File Watcher Validation (MEDIUM)
**Action:**
```bash
cd seeder
make build
./bin/seeder start
# In another terminal:
cp ../test-package/hello-world@1.0.0.tgz ./packages/
# Observe logs
```

**Success Criteria:**
- Package auto-detected within 2 seconds
- Successfully added to seeder
- File moved to `packages/seeded/`
- Appears in `./bin/seeder list`

**Time Estimate:** 5 minutes

---

## 🔍 Known Issues to Address

### Issue 1: CLI Terminology Audit
**Status:** ⚠️ Minor inconsistencies remain  
**Impact:** Low (cosmetic)

**Files to Review:**
- `seeder/internal/cli/add_package.go:147` - "Publisher" in output
- `README.md` - "For Publishers" heading
- Any help text still using "Publisher" instead of "Packager"

**Action:**
```bash
cd /home/fulgidus/Documents/libreseed
grep -r "Publisher" --include="*.go" --include="*.md" seeder/ | grep -v "DHT" | grep -v "protocol"
```

---

### Issue 2: Add Package Command Verification
**Status:** ⚠️ Command may not exist in seeder  
**Impact:** Medium (E2E test may fail at step 6)

**Current State:**
- E2E test script checks for `seeder add-package` command
- If not found, displays warning and continues
- May need to test validation differently

**Action:**
```bash
cd seeder
./bin/seeder --help | grep "add-package"
```

---

## 📊 Test Coverage Matrix

| Component | Unit Tests | Integration Tests | E2E Tests | Manual Tests |
|-----------|-----------|------------------|-----------|--------------|
| **Packager** | ✅ 100% | ✅ Available | ✅ Available | N/A |
| **Seeder - Validation** | ✅ Implemented | ⚠️ Not Run | ⚠️ Not Run | N/A |
| **Seeder - File Watcher** | N/A | ✅ Script Ready | N/A | ✅ Guide Available |
| **Seeder - DHT** | ✅ Implemented | ⚠️ Not Run | N/A | N/A |
| **Seeder - Torrent** | ✅ Implemented | ⚠️ Not Run | N/A | N/A |

**Legend:**
- ✅ Complete and passing
- ⚠️ Exists but not yet run
- ❌ Missing or failing
- N/A = Not applicable

---

## 🚀 Quick Start: Run All Tests

```bash
#!/bin/bash
# Run complete test suite

echo "=========================================="
echo "LibreSeed Complete Test Suite"
echo "=========================================="
echo ""

# Test 1: File Watcher
echo "[1/2] Running File Watcher Integration Tests..."
cd seeder
chmod +x test-watcher.sh
./test-watcher.sh
WATCHER_RESULT=$?

# Test 2: End-to-End Package Flow
echo ""
echo "[2/2] Running End-to-End Package Flow Tests..."
cd ../test-e2e
chmod +x run-e2e-test.sh
./run-e2e-test.sh
E2E_RESULT=$?

# Summary
echo ""
echo "=========================================="
echo "Test Suite Summary"
echo "=========================================="
echo "File Watcher Tests: $([ $WATCHER_RESULT -eq 0 ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "E2E Package Tests:  $([ $E2E_RESULT -eq 0 ] && echo "✅ PASS" || echo "❌ FAIL")"
echo ""

if [ $WATCHER_RESULT -eq 0 ] && [ $E2E_RESULT -eq 0 ]; then
    echo "✅ All tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
```

**Save as:** `run-all-tests.sh` in project root

---

## 📋 Test Results Template

```markdown
# Test Run Results

**Date:** YYYY-MM-DD  
**Tester:** [Name]  
**Branch/Commit:** [commit hash]

## File Watcher Tests
- [ ] Test 1: Seeder startup - PASS/FAIL
- [ ] Test 2: Package detection - PASS/FAIL
- [ ] Test 3: File movement - PASS/FAIL
- [ ] Test 4: Invalid handling - PASS/FAIL
- [ ] Test 5: Multiple files - PASS/FAIL
- [ ] Test 6: Status verification - PASS/FAIL
- [ ] Test 7: Graceful shutdown - PASS/FAIL

**Overall:** PASS/FAIL

## E2E Package Flow Tests
- [ ] Keypair generation - PASS/FAIL
- [ ] Package creation - PASS/FAIL
- [ ] Inspection - PASS/FAIL
- [ ] Minimal manifest - PASS/FAIL
- [ ] Seeder validation - PASS/FAIL/SKIP

**Overall:** PASS/FAIL

## Issues Found
[List any issues encountered]

## Notes
[Additional observations]
```

---

## 🔧 Troubleshooting Guide

### File Watcher Tests Fail
**Symptoms:** Files not detected, watcher not starting  
**Solutions:**
1. Check `seeder.yaml` has `manifest.watch_dir: "./packages"`
2. Ensure `./packages` directory exists and writable
3. Check logs for "File watcher started successfully"
4. Verify test package is valid: `tar -tzf test-file.tar.gz`

### E2E Tests Fail at Package Creation
**Symptoms:** Packager fails, no tarball created  
**Solutions:**
1. Rebuild packager: `cd packager && make build`
2. Check test project exists: `test-e2e/test-project/`
3. Verify key generation worked
4. Check for detailed error in packager output

### Seeder Won't Start
**Symptoms:** Seeder exits immediately  
**Solutions:**
1. Check `seeder.yaml` configuration
2. Ensure DHT port not in use: `lsof -i :6881`
3. Check logs for specific error
4. Try with minimal config first

---

## 📈 Next Steps After Testing

### If All Tests Pass ✅
1. Update `IMPLEMENTATION_STATUS.md` with test results
2. Tag release: `v1.3.0-beta`
3. Update README with installation instructions
4. Document any configuration changes needed
5. Plan deployment testing

### If Tests Fail ❌
1. Document failures in GitHub issues
2. Prioritize critical failures
3. Fix and retest
4. Update test scripts if false positives found

---

## 📞 Test Execution Support

**For automated test execution, run:**
```bash
# File Watcher Test
cd seeder && ./test-watcher.sh

# E2E Package Flow Test
cd test-e2e && ./run-e2e-test.sh

# Both (if run-all-tests.sh created)
./run-all-tests.sh
```

**For manual validation, follow:**
- `seeder/FILE_WATCHER_TEST.md` - Step-by-step manual tests
- `test-e2e/E2E-TEST-INSTRUCTIONS.md` - Manual E2E workflow

---

**Status: Ready for Test Execution** 🚀
