# DHT Refactoring Testing Guide

## Overview
This document provides testing instructions for the DHT port conflict resolution refactoring.

**Goal:** Verify that the seeder now uses a single shared DHT server (eliminating port 6881 conflicts).

---

## Changes Summary

### Modified Files
1. **`internal/dht/manager.go`** - Now accepts external DHT server
2. **`internal/torrent/engine.go`** - Added `Client()` accessor method
3. **`internal/cli/start.go`** - Rewrote DHT initialization to share torrent engine's DHT

### Architecture Change
```
BEFORE (Port Conflict):
├─ Torrent Engine → DHT Server (port 6881)
└─ DHT Manager    → DHT Server (port 6881) ❌ CONFLICT

AFTER (Shared DHT):
└─ Torrent Engine → DHT Server (port 6881)
                    └─> DHT Manager (reuses same DHT)
```

---

## Testing Steps

### 1. Build Test

```bash
cd seeder
go build -v ./cmd/seeder
```

**Expected Result:** ✅ Clean build with no errors

**If build fails:**
- Check for import errors
- Verify Go version compatibility (≥1.21)
- Run `go mod tidy`

---

### 2. Static Analysis

```bash
# Run Go linter
go vet ./...

# Optional: Run golangci-lint
golangci-lint run
```

**Expected Result:** ✅ No critical errors

---

### 3. Configuration Check

Verify your `seeder.yaml` has DHT enabled:

```yaml
dht:
  enabled: true
  port: 6881  # This port is now ONLY used by torrent engine
```

---

### 4. Integration Test - Single DHT Server

#### A. Start the Seeder

```bash
./seeder start --config seeder.yaml
```

#### B. Check Logs for Success Indicators

Look for this sequence in the logs:

```
INFO  Initializing DHT manager (using torrent engine's DHT)
INFO  DHT manager started successfully (reusing torrent engine's DHT)
```

**❌ Should NOT see:**
```
ERROR bind: address already in use
ERROR Failed to start DHT manager
```

#### C. Verify Network Port Usage

In a separate terminal:

```bash
# Check UDP port 6881 usage
sudo netstat -tulpn | grep 6881

# OR use lsof
sudo lsof -i UDP:6881
```

**Expected Result:** 
```
udp    0    0 0.0.0.0:6881    0.0.0.0:*    12345/seeder
```

✅ **ONLY ONE process** should be listening on port 6881

---

### 5. Functional Test - DHT Operations

#### A. Add a Torrent

```bash
# In another terminal
./seeder add /path/to/test.torrent
```

#### B. Verify DHT Activity

Check logs for:
```
DEBUG DHT: announcing infohash
DEBUG DHT: peers discovered
```

#### C. Test Manifest Storage (if applicable)

If your setup includes manifest storage:

```bash
# Store a test manifest
./seeder manifest store <package-name> <manifest-data>

# Retrieve the manifest
./seeder manifest get <package-name>
```

**Expected Result:** ✅ Successful store/retrieve operations

---

### 6. Re-Announce Test

DHT manager should periodically re-announce manifests (every 22 hours per spec).

#### Verify Re-Announce Loop Started

Check logs:
```
DEBUG DHT re-announce cycle started
```

#### Optional: Trigger Immediate Re-Announce

If you have a debug endpoint or can modify code temporarily:

```go
// In manager.go, temporarily reduce ReannounceInterval for testing
const ReannounceInterval = 2 * time.Minute
```

Then rebuild and watch logs for re-announce activity.

---

### 7. Graceful Shutdown Test

```bash
# Send SIGTERM to seeder
kill -TERM <seeder-pid>

# OR use Ctrl+C
```

**Expected Log Sequence:**
```
INFO  Received shutdown signal
INFO  Stopping DHT manager
INFO  DHT manager stopped
INFO  Stopping torrent engine
INFO  Torrent engine stopped
```

✅ **No errors or panics during shutdown**

---

### 8. Edge Case Testing

#### A. Start Without DHT

```yaml
# seeder.yaml
dht:
  enabled: false
```

```bash
./seeder start --config seeder.yaml
```

**Expected:**
```
INFO  DHT disabled in configuration
```

✅ Seeder starts normally without DHT

#### B. DHT Extraction Failure (Simulated)

This tests the fallback logic if DHT server is not available.

**Check logs contain:**
```
WARN  Continuing without DHT - peer discovery will be limited
```

✅ Seeder continues operating (graceful degradation)

---

## Success Criteria

### ✅ All Tests Must Pass:

1. **Build Success:** Code compiles without errors
2. **Single Port Usage:** Only ONE process on UDP 6881
3. **Log Confirmation:** "DHT manager started successfully (reusing torrent engine's DHT)"
4. **No Port Conflicts:** No "address already in use" errors
5. **Functional DHT:** Torrent announces and peer discovery work
6. **Graceful Shutdown:** Clean shutdown with no errors
7. **Fallback Behavior:** Graceful degradation if DHT unavailable

---

## Troubleshooting

### Issue: Build Fails with Import Error

**Solution:**
```bash
go mod tidy
go clean -cache
go build -v ./cmd/seeder
```

---

### Issue: Still Seeing Port Conflict

**Check:**
1. Verify you rebuilt the binary: `go build -v ./cmd/seeder`
2. Kill any old seeder processes: `pkill -9 seeder`
3. Verify config has `dht.enabled: true`
4. Check if another application is using port 6881: `sudo lsof -i UDP:6881`

---

### Issue: "DHT server is not of expected type"

**Cause:** Anacrolix/torrent version mismatch or API change

**Solution:**
1. Check `go.mod` for `github.com/anacrolix/torrent` version
2. Verify `AnacrolixDhtServerWrapper` is available in that version
3. Run `go mod tidy` to ensure consistent dependencies

---

### Issue: No DHT Activity in Logs

**Check:**
1. DHT is enabled in config: `dht.enabled: true`
2. Firewall allows UDP 6881: `sudo ufw allow 6881/udp`
3. Check if bootstrap nodes are reachable
4. Verify logs for connection attempts

---

## Performance Verification

### Monitor Resource Usage

```bash
# CPU and Memory usage
top -p $(pgrep seeder)

# Network connections
netstat -an | grep 6881 | wc -l
```

**Expected:**
- Single UDP 6881 socket
- Comparable CPU/memory to previous version
- No memory leaks over time

---

## Regression Testing Checklist

Ensure existing functionality still works:

- [ ] Torrent adding/removing
- [ ] Peer discovery and connection
- [ ] File seeding and upload
- [ ] DHT announces for torrents
- [ ] CLI commands (add, list, remove, status)
- [ ] Config reload (if supported)
- [ ] Metrics/monitoring (if enabled)

---

## Code Review Checklist

For reviewers:

- [ ] `dht.Manager` no longer creates DHT server
- [ ] `dht.Manager.Stop()` does not close DHT server
- [ ] `torrent.Engine.Client()` method returns `*torrent.Client`
- [ ] `start.go` extracts DHT via type assertion
- [ ] Graceful degradation if DHT unavailable
- [ ] No breaking changes to public APIs
- [ ] Error handling comprehensive
- [ ] Logs clear and actionable

---

## Next Steps After Testing

### If All Tests Pass ✅

1. **Commit Changes:**
   ```bash
   git add seeder/internal/dht/manager.go
   git add seeder/internal/torrent/engine.go
   git add seeder/internal/cli/start.go
   git commit -m "fix: eliminate DHT port conflict by sharing single DHT server"
   ```

2. **Update Documentation:**
   - Update architecture diagrams
   - Add note about shared DHT in README
   - Document DHT configuration options

3. **Deploy:**
   - Build release binary
   - Test in staging environment
   - Roll out to production

### If Tests Fail ❌

1. Document the failure with:
   - Error messages
   - Log output
   - Steps to reproduce
   - Environment details

2. Create GitHub issue or notify maintainers

3. Rollback if necessary

---

## Contact

For questions or issues with this refactoring:
- Check project documentation
- Open GitHub issue
- Contact maintainers

---

**Last Updated:** 2025-01-XX
**Refactoring Author:** Project Manager Agent + Development Team
