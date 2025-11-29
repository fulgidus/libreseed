# LIBRESEED Protocol Specification v1.3 — Protocol Overview

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Core Architecture →](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md)

---

## 1. Protocol Overview

### 1.1 What is LibreSeed?

**LibreSeed is a fully decentralized P2P protocol for software package distribution.**

It is **NOT** an npm integration tool.  
It is **NOT** a gateway-centric system.  
It is a **protocol-first design** that enables zero-cost, censorship-resistant package distribution.

---

### 1.2 Core Principles

- ✅ **No central servers** — Pure P2P architecture
- ✅ **No HTTP/DNS dependencies** — Complete decentralization via DHT
- ✅ **Protocol-first** — Binaries before bridges
- ✅ **Zero cost** — No infrastructure required
- ✅ **Cryptographically secure** — Ed25519 signatures
- ✅ **Censorship-resistant** — No single point of failure
- ✅ **Self-sustaining** — Community-powered seeder network

---

### 1.3 Design Philosophy

**Primary Deliverables:**
1. `libreseed-packager` — CLI binary for creating and publishing packages
2. `libreseed-seeder` — Daemon binary for maintaining network availability
3. **Protocol specification** (this document)

**Secondary Deliverable:**
- NPM bridge/gateway (optional ecosystem integration layer)

**Storage Model:**
- Home directory symlinks: `~/.libreseed/packages/`
- Similar to pnpm content-addressable storage
- No `node_modules` pollution

---

**Navigation:**
[← INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Core Architecture →](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
