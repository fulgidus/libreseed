# File Watcher Feature - Critical Issues Report

**Date:** 2025-11-29  
**Feature:** Automatic Package File Watcher  
**Status:** ⚠️ **BLOCKING ISSUES IDENTIFIED**  
**Priority:** HIGH

---

## Issue #1: Invalid Package Handling Not Implemented

**Severity:** 🔴 **HIGH** (Data Integrity Risk)  
**Status:** Open  
**Affects:** Production readiness

### Description
The file watcher does not validate package manifest format before processing. Invalid packages are silently ignored and not moved to the `invalid/` directory as designed.

### Expected Behavior
1. Validate package manifest format before adding to seeder
2. If invalid, log clear error message
3. Move invalid package to `packages/invalid/` directory
4. Continue monitoring for new files

### Actual Behavior
- No manifest validation performed
- Invalid packages appear to be silently ignored
- `packages/invalid/` directory remains empty
- No error logs generated
- Invalid files may accumulate in watch directory

### Impact
- **User Experience:** No feedback when dropping invalid packages
- **Operations:** Invalid files clutter watch directory
- **Audit:** No record of rejected packages
- **Data Integrity:** Potential for processing corrupted data

### Evidence (Test 4 Results)
```
[Test 4] Testing invalid package handling...
✗ FAIL: Invalid package not detected as error
✗ FAIL: Invalid file not moved to invalid/

packages/
├── invalid/  (empty) ❌ Expected: invalid-test.tar.gz
└── seeded/   (previous valid packages)
```

### Root Cause
Location: `seeder/internal/watcher/watcher.go`, function `processPackage()`

Missing validation step before calling `addPackage()`:
1. Extract/peek at manifest
2. Validate required fields
3. Check format compliance
4. Handle errors appropriately

### Reproduction Steps
1. Start seeder with file watcher
2. Copy invalid package to `packages/` directory
3. Wait 5 seconds
4. Observe: No error logged, file not moved to `invalid/`

### Recommended Fix

```go
// In watcher.go, processPackage() function
func (w *FileWatcher) processPackage(filename string) {
    packagePath := filepath.Join(w.config.WatchDir, filename)
    
    // ADD: Validate manifest before processing
    if err := w.validatePackageManifest(packagePath); err != nil {
        w.logger.Error("invalid package manifest", 
            zap.String("file", filename),
            zap.Error(err))
        w.moveToInvalid(filename)
        return
    }
    
    // Existing addPackage() call...
}

// ADD: New validation function
func (w *FileWatcher) validatePackageManifest(packagePath string) error {
    // Extract manifest from tarball
    // Parse and validate required fields
    // Return error if invalid
}

// ADD: New move-to-invalid function
func (w *FileWatcher) moveToInvalid(filename string) error {
    src := filepath.Join(w.config.WatchDir, filename)
    dest := filepath.Join(w.invalidDir, filename)
    return os.Rename(src, dest)
}
```

### Testing Requirements
- Unit tests for manifest validation logic
- Integration test for invalid package handling
- Test various invalid formats (missing fields, corrupt tarball, empty files)

---

## Issue #2: Test Suite Timeout on Concurrent File Processing

**Severity:** 🟠 **MEDIUM-HIGH** (Potential Deadlock/Race Condition)  
**Status:** Open  
**Affects:** Reliability under load

### Description
Test 5 (multiple file processing) hangs indefinitely when 3 files are added simultaneously to the watch directory. Test times out after 120 seconds without completion.

### Expected Behavior
1. Multiple files added to watch directory
2. Watcher processes all files concurrently
3. All files moved to appropriate directories
4. Test completes within 30 seconds

### Actual Behavior
- Test initiates 3-file copy
- Process hangs indefinitely
- No progress after initial operations
- Timeout after 120+ seconds
- Requires manual process termination

### Impact
- **Reliability:** Potential deadlock under concurrent load
- **Testing:** Cannot verify multi-file handling
- **Production:** Risk of service hang with multiple simultaneous drops

### Evidence (Test 5 Results)
```
[Test 5] Testing multiple file processing...
ℹ Adding 3 files simultaneously...

[Process hangs - no output for 120+ seconds]
(Command timed out after 120000 ms)
```

### Possible Root Causes

#### Theory 1: Deadlock in Goroutine Synchronization
- Multiple `processPackage()` goroutines running
- Potential lock contention on shared resources
- Mutex deadlock or channel blocking

#### Theory 2: File System Event Storm
- Rapid file operations trigger event flood
- Processing queue backs up
- Debounce timer conflicts

#### Theory 3: Test Script Logic Error
- Test waits for condition that never occurs
- Incorrect file count or status check
- Race between file operations and assertions

### Recommended Investigation

1. **Add Debug Logging**
   ```go
   // In watcher.go
   func (w *FileWatcher) processPackage(filename string) {
       w.logger.Debug("processPackage: START", zap.String("file", filename))
       defer w.logger.Debug("processPackage: END", zap.String("file", filename))
       // existing code...
   }
   ```

2. **Review Goroutine Management**
   - Check for unbuffered channels
   - Verify WaitGroup usage
   - Look for select statements without timeout

3. **Test Script Isolation**
   - Run minimal concurrent test manually
   - Verify test wait conditions
   - Check for race conditions in test logic

4. **Profile Under Load**
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/goroutine
   ```

### Reproduction Steps
1. Start seeder with file watcher
2. Copy 3+ files simultaneously to `packages/`
3. Observe process behavior
4. Check if all files are processed or if process hangs

---

## Issue #3: Test Assertion Logic Errors (Test 2)

**Severity:** 🟡 **MEDIUM** (False Negative in Test Suite)  
**Status:** Open  
**Affects:** Test reliability

### Description
Test 2 reports failure despite feature working correctly. Logs show successful package detection and processing, but test validation checks fail.

### Evidence

**Test Result:**
```
[Test 2] Testing automatic package detection...
✗ FAIL: Package not detected
✗ FAIL: Package not added to seeder
```

**Actual Logs (prove it worked):**
```
INFO watcher/watcher.go:213 processing package {"file": "test-auto-1.tar.gz"}
INFO torrent-engine torrent/engine.go:699 added package as torrent 
     {"info_hash": "11d79e7e786eebaf37c89984d6306dc804da45e9"}
INFO watcher/watcher.go:254 package successfully added to seeder
DEBUG watcher/watcher.go:275 package moved to seeded directory
```

### Root Cause
Location: `test-watcher.sh`, Test 2 validation logic

Likely issues:
- Timing problem (checks too early)
- Wrong validation method (checking wrong location/status)
- Incorrect success criteria

### Recommended Fix
Review test script assertions:
```bash
# Current (failing) approach
# Need to review lines ~110-120 of test-watcher.sh

# Suggested approach: Parse logs for success indicators
if grep -q "package successfully added to seeder" test-watcher.log; then
    pass "Package added to seeder"
else
    fail "Package not added to seeder"
fi
```

---

## Issue #4: Test Script Path Configuration (RESOLVED)

**Severity:** 🔴 **CRITICAL** (Blocking all tests)  
**Status:** ✅ **RESOLVED**  

### Description
Test script referenced incorrect binary path causing immediate test failure.

### Fix Applied
```bash
# Before: ./bin/seeder
# After:  ./build/seeder
sed -i 's|./bin/seeder|./build/seeder|g' test-watcher.sh
```

### Verification
```
✓ PASS: Seeder binary exists
✓ PASS: Seeder started (PID: 374556)
```

---

## Priority Action Items

### 🔴 Critical (Must Fix Before Merge)
1. **Issue #1** - Implement invalid package handling
   - Required for production readiness
   - Data integrity risk
   - User experience impact

### 🟠 High (Should Fix Before Merge)
2. **Issue #2** - Debug and fix concurrent processing timeout
   - Potential reliability issue
   - Blocks full test validation
   - May indicate serious bug

### 🟡 Medium (Fix Soon)
3. **Issue #3** - Correct test assertion logic
   - False negatives reduce test value
   - May hide real issues
   - Quick fix, high value

---

## Testing Blockers

The following acceptance criteria **cannot be verified** until issues are resolved:

- [ ] Invalid packages moved to `invalid/` directory (**Issue #1**)
- [ ] Multiple concurrent files processed correctly (**Issue #2**)
- [ ] Status command shows correct package count (**Issue #2** - test incomplete)
- [ ] Graceful shutdown works without errors (**Issue #2** - test skipped)

---

## Recommendations

### Immediate Actions
1. **Implement manifest validation** (Issue #1)
   - Highest priority for production readiness
   - Clear specification available
   - Should take 2-4 hours to implement and test

2. **Debug timeout** (Issue #2)
   - Add comprehensive debug logging
   - Test manually with simplified scenario
   - Profile for deadlocks
   - Estimated: 4-8 hours investigation

3. **Fix test assertions** (Issue #3)
   - Review and correct validation logic
   - Quick win for test reliability
   - Estimated: 1-2 hours

### Decision Point
**Should this feature be merged?**

**Recommendation:** ❌ **NO - Do not merge**

**Rationale:**
- Issue #1 is a **critical gap** in designed functionality
- Issue #2 suggests **potential deadlock** that could affect production
- 47% test failure rate unacceptable for merge
- Core functionality works, but edge cases are broken

**Path Forward:**
1. Fix Issue #1 (invalid handling) - **MUST DO**
2. Investigate Issue #2 (timeout) - **MUST DO**
3. Fix Issue #3 (test assertions) - **SHOULD DO**
4. Re-run full test suite
5. If all green, approve merge

---

## Additional Notes

### What's Working Well
- Core file detection and processing ✅
- File organization and movement ✅
- Integration with torrent engine ✅
- Startup and initialization ✅
- Clean logging and observability ✅

### What Needs Work
- Error handling for invalid packages ❌
- Concurrent file processing reliability ❌
- Test suite completeness ❌

---

**Report Author:** White Box Testing Agent  
**Escalation:** Issues #1 and #2 require @developer attention  
**Timeline:** Recommend 1-2 day delay for fixes before merge approval
