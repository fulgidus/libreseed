# LibreSeed

**Decentralized software distribution system using BitTorrent DHT**

LibreSeed is a modern solution for peer-to-peer software package distribution, leveraging the BitTorrent DHT (Distributed Hash Table) to ensure availability, resilience, and decentralization.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Developer Guide](#developer-guide)
- [Architecture](#architecture)
- [License](#license)

---

## Features

✅ **Decentralized** — No central server, discovery through BitTorrent DHT  
✅ **Resilient** — Peer-to-peer distribution with automatic redundancy  
✅ **Modern CLI** — Intuitive command-line interface for daemon management  
✅ **Robust Daemon** — Background service with graceful shutdown  
✅ **Monitoring** — Real-time statistics and system status  
✅ **Full Automation** — Makefile with 20+ targets for build, test, release  

---

## Quick Start

### Installation

#### Binary Installation (Recommended)

Install the latest release directly from GitHub:

```bash
# User installation (no sudo required, installs to ~/.local/bin)
curl -fsSL https://raw.githubusercontent.com/fulgidus/libreseed/main/scripts/install.sh | bash

# System-wide installation (requires sudo, installs to /usr/local/bin)
curl -fsSL https://raw.githubusercontent.com/fulgidus/libreseed/main/scripts/install.sh | bash -s -- --system
```

**Features:**
- ✅ Automatic platform/architecture detection (Linux, macOS, Windows)
- ✅ Downloads latest release from GitHub
- ✅ SHA256 checksum verification (mandatory)
- ✅ Installs `lbs` and `lbsd` binaries
- ✅ No build dependencies required

**Alternative: Manual Binary Installation**

1. Download the latest release for your platform from [Releases](https://github.com/fulgidus/libreseed/releases)
2. Verify the checksum:
   ```bash
   sha256sum -c lbs-linux-amd64.sha256
   ```
3. Make executable and move to PATH:
   ```bash
   chmod +x lbs-linux-amd64
   sudo mv lbs-linux-amd64 /usr/local/bin/lbs
   ```

#### Build from Source

If you prefer to build from source or need the latest development version:

**Prerequisites:**
- **Go** 1.21 or higher
- **Make** (for build automation)
- **Git** (to clone the repository)

```bash
# Clone the repository
git clone https://github.com/fulgidus/libreseed.git
cd libreseed

# Install from source
./scripts/install-from-source.sh
```

The build script performs:
- Prerequisites verification (Go, Make, sha256sum)
- Binary builds (`lbs`, `lbsd`)
- Checksum generation and verification
- Installation to `/usr/local/bin` (requires sudo)
- Data directory creation in `~/.local/share/libreseed`

### Basic Usage

```bash
# Start the daemon
lbs start

# Check status
lbs status

# Show statistics
lbs stats

# Stop the daemon
lbs stop

# Restart the daemon
lbs restart

# Show version
lbs version
```
