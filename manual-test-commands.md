# Manual DHT Testing Commands

If you prefer to test manually, follow these steps:

## 1. Install Dependencies
```bash
cd /home/fulgidus/Documents/libreseed
go get github.com/anacrolix/dht/v2
go mod tidy
```

## 2. Build the Daemon
```bash
mkdir -p bin
go build -o bin/libreseed-daemon ./cmd/libreseed-daemon
```

## 3. Create Data Directory
```bash
mkdir -p data
```

## 4. Start the Daemon
```bash
./bin/libreseed-daemon start --config config.test.yaml
```

The daemon will start and output logs. Wait for messages indicating DHT initialization.

## 5. Test DHT Endpoints (in another terminal)

### Test DHT Statistics
```bash
curl http://localhost:8080/dht/stats | jq
```

Expected response:
```json
{
  "routing_table": {
    "nodes": 0,
    "good_nodes": 0,
    "bad_nodes": 0
  },
  "queries_sent": 0,
  "responses_received": 0,
  "announces_sent": 0
}
```

### Test DHT Announcements
```bash
curl http://localhost:8080/dht/announcements | jq
```

Expected response:
```json
{
  "packages": []
}
```

### Test DHT Peers
```bash
curl http://localhost:8080/dht/peers | jq
```

Expected response:
```json
{
  "peers": []
}
```

### Test DHT Discovery
```bash
curl http://localhost:8080/dht/discovery | jq
```

Expected response:
```json
{
  "cached_results": [],
  "statistics": {
    "total_cached": 0,
    "cache_hits": 0,
    "cache_misses": 0
  }
}
```

## 6. Stop the Daemon
```bash
./bin/libreseed-daemon stop
```

Or press `Ctrl+C` in the terminal where it's running.

## Troubleshooting

### Compilation Errors
If you get compilation errors about missing packages:
```bash
go mod download
go mod verify
```

### Port Already in Use
If port 8080 or 6881 is already in use, modify `config.test.yaml`:
```yaml
listen_address: "0.0.0.0:8081"  # Change to different port
dht_port: 6882                   # Change to different port
```

### DHT Not Responding
- Wait 30-60 seconds after starting for DHT bootstrap to complete
- Check that UDP port 6881 is not blocked by firewall
- Verify bootstrap nodes are reachable

### Empty DHT Stats
This is normal on initial startup. DHT nodes and stats will populate as:
- The daemon connects to bootstrap nodes
- Other peers discover this node
- Packages are announced

## Next Steps After Successful Testing

Once DHT is working, you can:
1. Add packages and verify they're announced
2. Monitor DHT node count growth
3. Test peer discovery with multiple instances
4. Implement package search functionality
5. Add more DHT endpoints for advanced features
