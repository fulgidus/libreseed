#!/bin/bash
# LibreSeed DHT Integration Test Script

set -e

echo "=== LibreSeed DHT Integration Test ==="
echo

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: Install dependencies
echo -e "${YELLOW}[1/5] Installing dependencies...${NC}"
cd /home/fulgidus/Documents/libreseed
go get github.com/anacrolix/dht/v2
go mod tidy
echo -e "${GREEN}✓ Dependencies installed${NC}"
echo

# Step 2: Build the daemon
echo -e "${YELLOW}[2/5] Building libreseed-daemon...${NC}"
mkdir -p bin
go build -o bin/libreseed-daemon ./cmd/libreseed-daemon
echo -e "${GREEN}✓ Build successful${NC}"
echo

# Step 3: Create data directory
echo -e "${YELLOW}[3/5] Creating data directory...${NC}"
mkdir -p data
echo -e "${GREEN}✓ Data directory ready${NC}"
echo

# Step 4: Start the daemon (in background)
echo -e "${YELLOW}[4/5] Starting libreseed-daemon...${NC}"
./bin/libreseed-daemon start --config config.test.yaml &
DAEMON_PID=$!
echo "Daemon started with PID: $DAEMON_PID"
echo "Waiting 5 seconds for DHT initialization..."
sleep 5
echo -e "${GREEN}✓ Daemon running${NC}"
echo

# Step 5: Test DHT endpoints
echo -e "${YELLOW}[5/5] Testing DHT endpoints...${NC}"
echo

# Test DHT Stats
echo "Testing GET /dht/stats"
curl -s http://localhost:8080/dht/stats | jq . || echo -e "${RED}Failed${NC}"
echo

# Test DHT Announcements
echo "Testing GET /dht/announcements"
curl -s http://localhost:8080/dht/announcements | jq . || echo -e "${RED}Failed${NC}"
echo

# Test DHT Peers
echo "Testing GET /dht/peers"
curl -s http://localhost:8080/dht/peers | jq . || echo -e "${RED}Failed${NC}"
echo

# Test DHT Discovery
echo "Testing GET /dht/discovery"
curl -s http://localhost:8080/dht/discovery | jq . || echo -e "${RED}Failed${NC}"
echo

echo -e "${GREEN}=== All tests completed ===${NC}"
echo
echo "The daemon is still running with PID: $DAEMON_PID"
echo "To stop it, run: kill $DAEMON_PID"
echo "Or use: ./bin/libreseed-daemon stop"
