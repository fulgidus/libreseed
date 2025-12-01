# File Watcher Integration - Testing Guide

## ✅ Integration Complete!

The file watcher has been successfully integrated into the seeder CLI.

### Files Modified/Created:

1. **`seeder/internal/config/config.go`** ✅
   - Added `WatchDir string` field to `ManifestConfig` (line 59-63)
   - Added default value `"./packages"` (line 100-103)
   - Added Viper configuration binding (line 174-177)

2. **`seeder/internal/watcher/watcher.go`** ✅ NEW (315 lines)
   - Full-featured file watcher implementation
   - Watches for `.tar.gz` and `.tgz` files
   - 2-second debouncing after file write completion
   - Automatic file movement after processing

3. **`seeder/internal/cli/start.go`** ✅
   - Import added (line 19)
   - Watcher initialization (lines 175-196)
   - Watcher shutdown (lines 219-227)

4. **Directory Structure Created:**
   ```
   seeder/
   └── packages/         # Watch directory (auto-created)
       ├── seeded/       # Successfully processed packages
       └── invalid/      # Failed validation packages
   ```

---

## 🧪 Testing Instructions

### Prerequisites:

1. **Build the seeder:**
   ```bash
   cd /home/fulgidus/Documents/libreseed/seeder
   make build
   # or
   go build -o bin/seeder ./cmd/seeder
   ```

2. **Prepare a test package:**
   ```bash
   # Option 1: Use existing test package
   cp /home/fulgidus/Documents/libreseed/test-package/hello-world@1.0.0.tgz ./test-file.tar.gz
   
   # Option 2: Create from examples
   cd /home/fulgidus/Documents/libreseed/examples/hello-world
   tar -czf /home/fulgidus/Documents/libreseed/seeder/test-file.tar.gz .
   ```

### Test 1: Basic File Watching

1. **Start the seeder:**
   ```bash
   cd /home/fulgidus/Documents/libreseed/seeder
   ./bin/seeder start
   ```

2. **Expected startup logs:**
   ```
   INFO    Initializing file watcher       {"watch_dir": "./packages"}
   INFO    File watcher started successfully
   INFO    Seeder service started successfully
   ```

3. **In another terminal, drop a test package:**
   ```bash
   cd /home/fulgidus/Documents/libreseed/seeder
   cp test-file.tar.gz ./packages/
   ```

4. **Expected watcher logs:**
   ```
   INFO    Processing package      {"file": "test-file.tar.gz"}
   INFO    Successfully added package to seeder    {"file": "test-file.tar.gz"}
   INFO    Moving processed file   {"from": "./packages/test-file.tar.gz", "to": "./packages/seeded/test-file.tar.gz"}
   ```

5. **Verify file moved:**
   ```bash
   ls -la ./packages/seeded/
   # Should show: test-file.tar.gz
   ```

6. **Check torrent stats:**
   ```bash
   ./bin/seeder status
   # Should show: 1 active torrent
   ```

### Test 2: Invalid Package Handling

1. **Create invalid file:**
   ```bash
   echo "not a valid tarball" > ./packages/invalid-package.tar.gz
   ```

2. **Expected watcher logs:**
   ```
   INFO    Processing package      {"file": "invalid-package.tar.gz"}
   ERROR   Failed to add package to seeder {"file": "invalid-package.tar.gz", "error": "..."}
   WARN    Moving invalid file     {"from": "./packages/invalid-package.tar.gz", "to": "./packages/invalid/invalid-package.tar.gz"}
   ```

3. **Verify file moved to invalid:**
   ```bash
   ls -la ./packages/invalid/
   # Should show: invalid-package.tar.gz
   ```

### Test 3: Multiple Files

1. **Copy multiple packages at once:**
   ```bash
   for i in {1..3}; do
       cp test-file.tar.gz ./packages/test-$i.tar.gz
       sleep 0.5  # Small delay between copies
   done
   ```

2. **Verify all processed:**
   ```bash
   ls -la ./packages/seeded/
   # Should show: test-1.tar.gz, test-2.tar.gz, test-3.tar.gz
   ```

### Test 4: Duplicate Detection

1. **Try to add same package twice:**
   ```bash
   cp ./packages/seeded/test-file.tar.gz ./packages/duplicate.tar.gz
   ```

2. **Expected log:**
   ```
   INFO    Processing package      {"file": "duplicate.tar.gz"}
   WARN    Package already exists in seeder        {"file": "duplicate.tar.gz"}
   INFO    Moving processed file   {"from": "./packages/duplicate.tar.gz", "to": "./packages/seeded/duplicate.tar.gz"}
   ```
   
   *(File still moved to seeded/ even if already tracked)*

### Test 5: Large File (Debouncing)

1. **Copy a large file to simulate slow write:**
   ```bash
   # Create a ~10MB file
   dd if=/dev/urandom of=./packages/large-test.tar.gz bs=1M count=10
   ```

2. **Expected behavior:**
   - Watcher detects file creation
   - Waits 2 seconds after last write event
   - Then processes the file

### Test 6: Graceful Shutdown

1. **While watching, press Ctrl+C**

2. **Expected shutdown logs:**
   ```
   INFO    Received shutdown signal        {"signal": "interrupt"}
   INFO    Shutting down seeder service...
   INFO    Stopping file watcher...
   INFO    File watcher stopped successfully
   INFO    Stopping DHT manager...
   INFO    DHT manager stopped successfully
   INFO    Seeder service stopped
   ```

### Test 7: Disabled Watcher

1. **Edit config to disable watcher:**
   ```yaml
   # seeder/seeder.yaml
   manifest:
     watch_dir: ""  # Empty = disabled
   ```

2. **Start seeder:**
   ```bash
   ./bin/seeder start
   ```

3. **Expected log:**
   ```
   INFO    File watcher disabled (no watch directory configured)
   ```

---

## 🔍 Troubleshooting

### Issue: "Failed to create file watcher"

**Cause:** Watch directory doesn't exist or insufficient permissions

**Solution:**
```bash
mkdir -p ./packages/seeded ./packages/invalid
chmod 755 ./packages
```

### Issue: "Failed to add package to seeder"

**Possible causes:**
1. Invalid tarball format → Check with `tar -tzf file.tar.gz`
2. Missing manifest → Package must contain a valid manifest
3. Duplicate infohash → Package already seeded

**Check logs for specific error**

### Issue: Files not being detected

**Possible causes:**
1. File extension not `.tar.gz` or `.tgz`
2. File written too quickly (debounce still waiting)
3. Watcher stopped unexpectedly

**Check:**
```bash
# View current torrents
./bin/seeder list

# Check watcher status in logs
grep "watcher" seeder.log
```

---

## 📊 Success Criteria

All tests should demonstrate:

- ✅ File detection within 2 seconds of write completion
- ✅ Successful package addition to seeder
- ✅ Correct file movement to `seeded/` or `invalid/`
- ✅ Duplicate detection and handling
- ✅ Clean shutdown without errors
- ✅ Proper logging at each stage

---

## 🔧 Configuration Reference

```yaml
# seeder.yaml
manifest:
  watch_dir: "./packages"  # Directory to watch (empty = disabled)
```

**Environment variable:**
```bash
export MANIFEST_WATCH_DIR="./custom-packages"
```

**CLI flag:**
```bash
# Currently not exposed as CLI flag, only via config file
```

---

## 📈 Next Steps After Testing

1. **Performance testing** with many concurrent file additions
2. **Integration with manifest loader** (when implemented)
3. **Metrics collection** for watcher operations
4. **Health endpoint** to expose watcher status
5. **Subdirectory watching** (if needed in future)

---

## 🐛 Known Limitations

1. **No recursive watching** - Only monitors top-level directory
2. **File extension hardcoded** - Only `.tar.gz` and `.tgz`
3. **No file size limit** - Will attempt to process any size
4. **Sequential processing** - One file at a time (by design)
5. **No automatic retry** - Failed files stay in `invalid/`

These are intentional design choices for simplicity and safety.

---

## 📝 Code Quality Notes

- **Error handling:** All errors logged, watcher continues on failure
- **Thread safety:** Mutex-protected debounce timer map
- **Resource cleanup:** Proper timer cleanup and goroutine shutdown
- **Testability:** All key functions are testable units
- **Logging:** Comprehensive logging for debugging
- **Graceful degradation:** Seeder continues if watcher fails

---

**Integration Status: ✅ COMPLETE AND READY FOR TESTING**

Last updated: 2025-01-XX
