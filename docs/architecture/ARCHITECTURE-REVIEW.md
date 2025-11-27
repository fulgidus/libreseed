# LibreSeed Architecture Review
**Version:** 1.0  
**Date:** 2025-01-27  
**Specification Version:** LIBRESEED-SPEC v1.1  
**Reviewer:** Architect Agent

---

## Executive Summary

This document provides a comprehensive architectural review of the LibreSeed P2P package registry system. The review validates core architectural decisions, addresses critical open questions, identifies risks, and provides technical recommendations for implementation.

**Key Findings:**
- ✅ Core architecture is sound: DHT + BitTorrent + Ed25519 is a proven, scalable foundation
- ⚠️ Module resolution requires custom Node.js hooks or loader implementation
- ✅ Recommended stack: **js-libp2p (Kad-DHT) + WebTorrent** for mature, battle-tested libraries
- ⚠️ Publisher discovery needs multi-strategy approach (DHT + bootstrapping + reputation)
- 🔴 High-risk areas: Node.js module resolution compatibility, DHT pollution, bootstrap centralization

---

## 1. Architectural Validation

### 1.1 DHT Key Scheme Analysis

**Current Design:** `libreseed:<package-name>`

**Validation:** ✅ SOUND

**Rationale:**
- Simple, deterministic, predictable lookups
- Namespace collision prevention via `libreseed:` prefix
- Flat namespace appropriate for global package registry
- Compatible with Mainline DHT and Kad-DHT key structures

**Recommendation:** **ADOPT AS-IS** with one enhancement:

```
Key Format: libreseed:<scope>/<package-name>
Examples:
  - libreseed:lodash (unscoped)
  - libreseed:@babel/core (scoped)
  - libreseed:@myorg/utils (private scope)
```

**Rationale for Enhancement:**
- Maintains npm compatibility for scoped packages
- Enables organizational namespacing
- No DHT structural changes required (scope is just part of the string)

---

### 1.2 Seeder Priority System

**Current Design:** Publishers have priority; downloaders become seeders after download

**Validation:** ✅ SOUND with RECOMMENDATIONS

**Analysis:**

| Aspect | Assessment | Details |
|--------|-----------|---------|
| **Publisher Priority** | ✅ Correct | Ensures authentic source availability |
| **Downloader Seeding** | ✅ Good | Increases availability through mesh network |
| **Incentive Model** | ⚠️ Weak | No economic incentive for long-term seeding |
| **Publisher Burden** | ⚠️ High | Publishers must maintain seeders indefinitely |

**Recommendations:**

1. **Implement Seeder Reputation System:**
   ```
   DHT Value Extensions:
   {
     "infohash": "...",
     "publisher_sig": "...",
     "seeders": [
       {"peer": "...", "reputation": 0.95, "uptime": "99.2%"},
       {"peer": "...", "reputation": 0.87, "uptime": "94.1%"}
     ]
   }
   ```

2. **Add Seeder Incentives:**
   - Organizations can run public seeders for reputation/goodwill
   - Mirror networks (like Debian/Fedora mirrors)
   - Optional paid seeding services (out of protocol scope)

3. **Publisher Seeder Strategy:**
   - Publishers should run redundant seeders (3+ recommended)
   - Use distributed infrastructure (multi-region)
   - Gateway can cache popular packages to reduce publisher load

---

### 1.3 Security Model

**Current Design:** Ed25519 signatures on DHT values

**Validation:** ✅ EXCELLENT

**Strengths:**
- ✅ Strong cryptographic foundation (Ed25519 = 128-bit security)
- ✅ Prevents DHT poisoning (unsigned values rejected)
- ✅ Publisher identity verification
- ✅ Immutable package integrity via infohash

**Potential Enhancements:**

1. **Key Revocation Mechanism:**
   ```
   DHT Key: libreseed:revocations
   Value: {
     "revoked_keys": [
       {"pubkey": "...", "reason": "compromised", "timestamp": "..."}
     ],
     "sig": "..."
   }
   ```

2. **Multi-Signature Support (Future):**
   - Enable organizational key management
   - Require N-of-M signatures for critical packages

3. **Transparency Log (Future):**
   - Append-only log of all package publications
   - Enables audit and anomaly detection

**Recommendation:** **ADOPT CURRENT + PLAN KEY REVOCATION**

---

### 1.4 Scalability Analysis

**DHT Scalability:**
- ✅ Mainline DHT handles 15-27 million concurrent nodes
- ✅ Kad-DHT (IPFS) handles millions of peers
- ✅ Flat key structure = O(log N) lookups
- ⚠️ Hot keys (popular packages) may experience churn

**BitTorrent Scalability:**
- ✅ Proven for massive file distribution (Linux ISOs, etc.)
- ✅ Mesh network scales with demand
- ✅ Built-in bandwidth optimization

**Bottlenecks:**
- 🔴 **Bootstrap nodes** (single point of failure)
- ⚠️ **Gateway** (potential centralization if poorly designed)
- ⚠️ **Publisher seeders** (must scale with package popularity)

**Mitigation:**
- Use federated bootstrap nodes (community-run)
- Gateway should be stateless, horizontally scalable
- Encourage mirror seeders for popular packages

---

## 2. Critical Open Questions (Section 13)

### 2.1 Publisher Discovery (Section 13.2)

**Question:** "How do gateways discover which publishers exist?"

**Problem Analysis:**
- DHT is optimized for key-value lookup, NOT enumeration
- No native "list all publishers" operation
- Enumeration would require full DHT traversal (infeasible)

**Recommended Solution: HYBRID DISCOVERY**

#### Strategy 1: Bootstrap Publisher Registry (Primary)

```json
// Maintained in well-known DHT key or published list
Key: libreseed:publishers
Value: {
  "publishers": [
    {
      "name": "npm-mirror",
      "pubkey": "ed25519:...",
      "packages": ["lodash", "express", "react"],
      "announce_url": "https://npm-mirror.libreseed.org/announce"
    },
    {
      "name": "rust-crates-mirror", 
      "pubkey": "ed25519:...",
      "packages_pattern": "rust:*"
    }
  ],
  "sig": "..."
}
```

**Pros:**
- Fast, efficient discovery
- Enables filtering by language/ecosystem
- Low DHT overhead

**Cons:**
- Requires centralized maintenance (or DAO/multi-sig governance)
- May exclude small/new publishers

#### Strategy 2: DHT Crawling (Supplementary)

```javascript
// Gateway periodically crawls DHT for libreseed: keys
async function discoverPublishers() {
  const knownKeys = await dht.crawl('libreseed:*'); // Prefix search
  const publishers = new Set();
  
  for (const key of knownKeys) {
    const value = await dht.get(key);
    if (value && value.publisher_pubkey) {
      publishers.add(value.publisher_pubkey);
    }
  }
  
  return Array.from(publishers);
}
```

**Pros:**
- Decentralized, no registry needed
- Discovers all active publishers

**Cons:**
- Computationally expensive
- Slow (hours for full DHT scan)
- DHT may not support prefix search natively

#### Strategy 3: Publisher Announcement (Supplementary)

```javascript
// Publishers announce themselves to well-known announce keys
Key: libreseed:announce:<date>
Value: {
  "announcements": [
    {"pubkey": "...", "packages": [...], "timestamp": "..."}
  ],
  "sig": "..."
}
```

**Pros:**
- Publishers control their visibility
- Time-based partitioning reduces key size

**Cons:**
- Requires publishers to actively announce
- Gateway must monitor multiple announce keys

**FINAL RECOMMENDATION:**

**Use Strategy 1 (Bootstrap Registry) as primary**, with:
- Community-governed registry (GitHub repo or DAO)
- Publishers submit PRs to add themselves
- Gateway caches registry locally
- **Strategy 3 (Announcement)** as fallback for discovery of new publishers

---

### 2.2 Module Resolution (Section 13.3)

**Question:** "How does Node.js resolve imports from `.libreseed_modules/`?"

**Problem Analysis:**
- Node.js hardcoded to search `node_modules/` directories
- Custom module paths require loader hooks or patches
- Must support: `require()`, `import`, TypeScript, bundlers (Webpack, Vite)

**Recommended Solution: NODE.JS LOADER HOOKS**

#### Implementation Path 1: Node.js --loader (ESM Only)

```javascript
// libreseed-loader.mjs
export async function resolve(specifier, context, nextResolve) {
  // Intercept bare imports (e.g., 'lodash')
  if (!specifier.startsWith('.') && !specifier.startsWith('/')) {
    const libreseedPath = path.join(
      findProjectRoot(context.parentURL),
      '.libreseed_modules',
      specifier
    );
    
    if (fs.existsSync(libreseedPath)) {
      return {
        url: pathToFileURL(libreseedPath).href,
        shortCircuit: true
      };
    }
  }
  
  return nextResolve(specifier, context);
}
```

**Usage:**
```bash
node --loader ./libreseed-loader.mjs app.js
```

**Pros:**
- Official Node.js API
- No patching required
- Supports ESM imports

**Cons:**
- ⚠️ ESM only (no `require()` support)
- ⚠️ Experimental API (may change)
- ⚠️ Requires `--loader` flag on every invocation

#### Implementation Path 2: NODE_PATH Environment Variable

```bash
export NODE_PATH=.libreseed_modules:$NODE_PATH
node app.js
```

**Pros:**
- Simple, no code required
- Works with CommonJS and ESM

**Cons:**
- 🔴 Flat resolution only (no nested `node_modules/`)
- 🔴 Doesn't support scoped packages well
- 🔴 User must set environment variable

#### Implementation Path 3: Custom require() Wrapper (CommonJS)

```javascript
// libreseed-require.js
const Module = require('module');
const originalResolveFilename = Module._resolveFilename;

Module._resolveFilename = function(request, parent, isMain) {
  // Try .libreseed_modules first
  if (!request.startsWith('.') && !request.startsWith('/')) {
    const libreseedPath = path.join(
      findProjectRoot(parent.filename),
      '.libreseed_modules',
      request
    );
    
    if (fs.existsSync(libreseedPath)) {
      return libreseedPath;
    }
  }
  
  return originalResolveFilename.call(this, request, parent, isMain);
};
```

**Usage:**
```javascript
// app.js
require('./libreseed-require'); // Must be first import
const lodash = require('lodash'); // Now resolves from .libreseed_modules/
```

**Pros:**
- Works with CommonJS `require()`
- No command-line flags needed

**Cons:**
- 🔴 Monkey-patching internals (fragile)
- 🔴 May break with Node.js updates
- 🔴 Doesn't work with ESM imports

#### Implementation Path 4: Symbolic Link (Workaround)

```bash
ln -s .libreseed_modules node_modules
```

**Pros:**
- No code changes needed
- Works with all module systems

**Cons:**
- 🔴 Conflicts with npm/yarn/pnpm
- 🔴 Git must ignore `node_modules/` or risks confusion
- 🔴 Windows requires admin privileges for symlinks

#### Implementation Path 5: Bundler Plugins (Production)

**Webpack:**
```javascript
// webpack.config.js
module.exports = {
  resolve: {
    modules: [
      path.resolve(__dirname, '.libreseed_modules'),
      'node_modules'
    ]
  }
};
```

**Vite:**
```javascript
// vite.config.js
export default {
  resolve: {
    alias: {
      // Map packages to .libreseed_modules
      'lodash': path.resolve(__dirname, '.libreseed_modules/lodash')
    }
  }
};
```

**Pros:**
- Clean, supported by bundlers
- Works in production builds

**Cons:**
- Requires bundler configuration
- Doesn't help with `node` CLI usage

---

**FINAL RECOMMENDATION: MULTI-PRONGED APPROACH**

1. **Development (Node.js CLI):**
   - Provide `libreseed` wrapper script that sets `NODE_PATH` or uses `--loader`
   ```bash
   libreseed run app.js  # Wraps: NODE_PATH=.libreseed_modules node app.js
   ```

2. **Development (Programmatic):**
   - Provide `libreseed/register` module to monkey-patch `require()`
   ```javascript
   require('libreseed/register');
   const lodash = require('lodash');
   ```

3. **Production (Bundlers):**
   - Provide Webpack/Vite/Rollup plugins
   - Document manual configuration

4. **Future (Node.js Core):**
   - Advocate for `package.json` field: `"modulePaths": [".libreseed_modules"]`
   - Propose Node.js enhancement request

**Implementation Priority:**
1. **Phase 1:** `NODE_PATH` wrapper + documentation (quick win)
2. **Phase 2:** `require()` monkey-patch for CommonJS
3. **Phase 3:** ESM `--loader` hook
4. **Phase 4:** Bundler plugins

---

### 2.3 DHT Implementation (Section 13.7)

**Question:** "Which DHT library should be used?"

**Analysis:**

| Library | Type | Language | Pros | Cons | Recommendation |
|---------|------|----------|------|------|----------------|
| **bittorrent-dht** | Mainline DHT | JavaScript | ✅ Battle-tested (BitTorrent)<br>✅ 15M+ node network<br>✅ Fast lookups | ⚠️ UDP-only (NAT issues)<br>⚠️ No Sybil resistance | ⚠️ ACCEPTABLE |
| **@libp2p/kad-dht** | Kademlia | JavaScript | ✅ Modern, maintained<br>✅ IPFS integration<br>✅ Multiple transports (TCP, WebSocket, WebRTC)<br>✅ NAT traversal | ⚠️ Smaller network than Mainline<br>⚠️ More complex | ✅ **RECOMMENDED** |
| **mainline** | Mainline DHT | Rust | ✅ High performance<br>✅ Memory safe | 🔴 Requires Node.js bindings<br>🔴 Additional complexity | ❌ NOT RECOMMENDED |

**RECOMMENDATION: `@libp2p/kad-dht` (js-libp2p)**

**Rationale:**

1. **Ecosystem Fit:**
   - JavaScript-native (no FFI/bindings)
   - Active IPFS community (millions of users)
   - Proven scalability

2. **Technical Advantages:**
   - **Multi-transport:** Fallback from WebSocket → TCP → WebRTC if UDP blocked
   - **NAT traversal:** Built-in hole-punching (critical for residential users)
   - **Content routing:** Designed for content-addressed systems (perfect fit)
   - **Peer routing:** Automatic peer discovery

3. **Implementation Example:**

```javascript
import { createLibp2p } from 'libp2p';
import { kadDHT } from '@libp2p/kad-dht';
import { noise } from '@chainsafe/libp2p-noise';
import { tcp } from '@libp2p/tcp';

const node = await createLibp2p({
  transports: [tcp()],
  streamMuxers: [mplex()],
  connectionEncryption: [noise()],
  dht: kadDHT({
    clientMode: false, // Gateway acts as full DHT node
    validators: {
      libreseed: (key, value) => {
        // Custom Ed25519 signature validation
        return verifyLibreseedValue(key, value);
      }
    }
  })
});

// Store package metadata
const key = '/libreseed/lodash';
const value = JSON.stringify({
  infohash: '...',
  publisher_pubkey: '...',
  timestamp: Date.now()
});
const signature = signEd25519(value, privateKey);

await node.contentRouting.put(
  Buffer.from(key),
  Buffer.from(JSON.stringify({ value, sig: signature }))
);

// Retrieve package metadata
const result = await node.contentRouting.get(Buffer.from(key));
const { value, sig } = JSON.parse(result.toString());
if (verifyEd25519(value, sig, publisherPubkey)) {
  const metadata = JSON.parse(value);
  // Proceed with torrent download
}
```

4. **Bootstrap Strategy:**

```javascript
const node = await createLibp2p({
  // ...
  peerDiscovery: [
    bootstrap({
      list: [
        '/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN',
        '/dnsaddr/libreseed-bootstrap-1.example.com/p2p/...',
        '/dnsaddr/libreseed-bootstrap-2.example.com/p2p/...'
      ]
    })
  ]
});
```

**Fallback Option:**
If libp2p proves too complex, use `bittorrent-dht` with custom NAT traversal logic.

---

### 2.4 Torrent Client (Section 13.8)

**Question:** "Which torrent client library should be used?"

**Analysis:**

| Library | Language | Pros | Cons | Recommendation |
|---------|----------|------|------|----------------|
| **WebTorrent** | JavaScript | ✅ Browser + Node.js<br>✅ Hybrid (WebRTC + TCP/UDP)<br>✅ 5M+ downloads/month<br>✅ Active maintenance | ⚠️ WebRTC overhead for Node.js-only | ✅ **RECOMMENDED** |
| **@ctrl/torrent** | TypeScript | ✅ TypeScript-native<br>✅ Modern API | ⚠️ Smaller community<br>⚠️ Less battle-tested | ⚠️ ACCEPTABLE (Alternative) |
| **transmission** | C | ✅ Extremely stable<br>✅ High performance | 🔴 Requires native bindings<br>🔴 CLI dependency | ❌ NOT RECOMMENDED |

**RECOMMENDATION: `webtorrent`**

**Rationale:**

1. **Proven Reliability:**
   - Used in production by WebTorrent Desktop (100K+ users)
   - Handles 10K+ concurrent torrents in testing
   - Active issue resolution

2. **Hybrid Network:**
   - TCP/UDP for traditional BitTorrent peers
   - WebRTC for browser peers (future-proofing)
   - Falls back gracefully if WebRTC unavailable

3. **Implementation Example:**

```javascript
import WebTorrent from 'webtorrent';

const client = new WebTorrent();

// Publisher: Seed package
const files = ['./lodash/'];
client.seed(files, (torrent) => {
  console.log('Infohash:', torrent.infoHash);
  
  // Store in DHT
  await dht.put(`libreseed:lodash`, {
    infohash: torrent.infoHash,
    publisher_pubkey: myPubkey,
    timestamp: Date.now()
  });
});

// Gateway: Download package
const dhtValue = await dht.get('libreseed:lodash');
const { infohash } = JSON.parse(dhtValue);

client.add(infohash, { path: './.libreseed_modules/lodash' }, (torrent) => {
  torrent.on('done', () => {
    console.log('Package downloaded to .libreseed_modules/lodash');
    
    // Become seeder (as per spec section 10)
    // WebTorrent automatically continues seeding
  });
});
```

4. **Performance Optimization:**

```javascript
const client = new WebTorrent({
  // Prioritize publisher seeders
  tracker: {
    announce: [
      'wss://tracker.webtorrent.dev',
      'wss://tracker.libreseed.org' // Custom tracker
    ]
  },
  
  // Seeder priority algorithm
  strategy: (peers) => {
    return peers.sort((a, b) => {
      // Prioritize peers matching publisher pubkey
      if (a.publisher && !b.publisher) return -1;
      if (!a.publisher && b.publisher) return 1;
      
      // Then by upload speed
      return b.uploadSpeed - a.uploadSpeed;
    });
  }
});
```

---

## 3. Architectural Risks

### 3.1 HIGH-RISK Areas

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|------------|
| **Bootstrap Node Centralization** | 🔴 CRITICAL | HIGH | Federated bootstrap nodes, fallback DNS seeds |
| **DHT Pollution** | 🔴 CRITICAL | MEDIUM | Ed25519 signatures, key expiration, reputation system |
| **Publisher Key Compromise** | 🔴 CRITICAL | LOW | Key revocation mechanism, multi-sig support |
| **Node.js Module Resolution** | 🟡 HIGH | HIGH | Provide multiple integration paths (wrapper, hooks, plugins) |
| **Gateway Centralization** | 🟡 HIGH | MEDIUM | Stateless design, encourage self-hosting |

### 3.2 MEDIUM-RISK Areas

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|------------|
| **NAT Traversal Failure** | 🟡 MEDIUM | MEDIUM | libp2p auto-relay, STUN/TURN servers |
| **Package Availability** | 🟡 MEDIUM | MEDIUM | Encourage mirror seeders, gateway caching |
| **Download Speed** | 🟡 MEDIUM | LOW | Multi-source downloads, CDN fallback |
| **DHT Lookup Latency** | 🟢 LOW | MEDIUM | Local DHT caching, aggressive prefetch |

### 3.3 Risk Mitigation Plan

#### Bootstrap Node Resilience

```javascript
const bootstrapStrategies = [
  // Strategy 1: Hardcoded bootstrap nodes
  { type: 'static', nodes: BOOTSTRAP_NODES },
  
  // Strategy 2: DNS seeds (like Bitcoin)
  { type: 'dns', domains: ['seed.libreseed.org'] },
  
  // Strategy 3: Cached peers from previous session
  { type: 'cache', path: '~/.libreseed/peers.json' },
  
  // Strategy 4: Fallback to IPFS bootstrap
  { type: 'ipfs', nodes: IPFS_BOOTSTRAP_NODES }
];

for (const strategy of bootstrapStrategies) {
  try {
    const peers = await resolvePeers(strategy);
    if (peers.length > 0) {
      await node.bootstrap(peers);
      break;
    }
  } catch (err) {
    console.warn(`Bootstrap strategy ${strategy.type} failed:`, err);
  }
}
```

#### DHT Pollution Prevention

```javascript
// Implement aggressive validation
const dhtValidators = {
  libreseed: (key, value) => {
    // 1. Verify signature
    const { value: data, sig } = JSON.parse(value);
    const metadata = JSON.parse(data);
    
    if (!verifyEd25519(data, sig, metadata.publisher_pubkey)) {
      throw new Error('Invalid signature');
    }
    
    // 2. Check timestamp (reject old values)
    const age = Date.now() - metadata.timestamp;
    if (age > 30 * 24 * 60 * 60 * 1000) { // 30 days
      throw new Error('Expired DHT value');
    }
    
    // 3. Verify publisher is in allowlist (optional)
    if (PUBLISHER_ALLOWLIST && !PUBLISHER_ALLOWLIST.includes(metadata.publisher_pubkey)) {
      throw new Error('Unknown publisher');
    }
    
    return true;
  }
};
```

---

## 4. Recommended System Architecture

### 4.1 High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         USER SPACE                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐      ┌──────────────┐                   │
│  │  Developer   │      │   Gateway    │                   │
│  │   Machine    │      │   (HTTPS)    │                   │
│  └──────┬───────┘      └──────┬───────┘                   │
│         │                      │                           │
│         │ libreseed install    │                           │
│         │ lodash               │                           │
│         └──────────────────────┘                           │
│                │                                            │
│                ▼                                            │
│  ┌──────────────────────────────────────┐                 │
│  │   LibreSeed Gateway (Node.js)        │                 │
│  │   - HTTP API (search, download)      │                 │
│  │   - DHT Client (libp2p/kad-dht)      │                 │
│  │   - Torrent Client (WebTorrent)      │                 │
│  │   - Signature Validator (Ed25519)    │                 │
│  │   - Cache Layer (optional)           │                 │
│  └─────────┬──────────────────┬─────────┘                 │
│            │                  │                            │
└────────────┼──────────────────┼────────────────────────────┘
             │                  │
    ┌────────▼────────┐    ┌───▼──────────┐
    │   DHT Network   │    │   BitTorrent │
    │   (libp2p)      │    │   Swarm      │
    │                 │    │   (WebTorrent)│
    │ - Store/Get     │    │              │
    │   package       │    │ - Seeders    │
    │   metadata      │    │ - Leechers   │
    └────────┬────────┘    └───┬──────────┘
             │                  │
    ┌────────▼────────────────── ▼────────────────┐
    │         PEER-TO-PEER NETWORK                │
    ├──────────────────────────────────────────────┤
    │                                              │
    │  ┌─────────────┐   ┌─────────────┐         │
    │  │  Publisher  │   │  Publisher  │         │
    │  │   Seeder    │   │   Seeder    │         │
    │  │  (Priority) │   │  (Priority) │         │
    │  └─────────────┘   └─────────────┘         │
    │                                              │
    │  ┌─────────────┐   ┌─────────────┐         │
    │  │  Downloader │   │  Downloader │         │
    │  │   Seeder    │   │   Seeder    │         │
    │  │  (Standard) │   │  (Standard) │         │
    │  └─────────────┘   └─────────────┘         │
    │                                              │
    └──────────────────────────────────────────────┘
```

### 4.2 Component Architecture

```
┌─────────────────────────────────────────────────────┐
│              LibreSeed Gateway                      │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────────────────────────────────────┐  │
│  │         HTTP API Layer                      │  │
│  │  - /search?q=lodash                         │  │
│  │  - /package/:name/download                  │  │
│  │  - /package/:name/metadata                  │  │
│  └────────────────┬────────────────────────────┘  │
│                   │                                │
│  ┌────────────────▼────────────────────────────┐  │
│  │      Business Logic Layer                   │  │
│  │  - Package Search                           │  │
│  │  - Version Resolution                       │  │
│  │  - Dependency Graph                         │  │
│  └────────────────┬────────────────────────────┘  │
│                   │                                │
│  ┌────────────────▼────────────────────────────┐  │
│  │         DHT Client                          │  │
│  │  - @libp2p/kad-dht                          │  │
│  │  - Bootstrap Management                     │  │
│  │  - Peer Discovery                           │  │
│  │  - Custom Validators                        │  │
│  └────────────────┬────────────────────────────┘  │
│                   │                                │
│  ┌────────────────▼────────────────────────────┐  │
│  │      Torrent Client                         │  │
│  │  - WebTorrent                               │  │
│  │  - Seeder Priority Logic                    │  │
│  │  - Multi-source Download                    │  │
│  └────────────────┬────────────────────────────┘  │
│                   │                                │
│  ┌────────────────▼────────────────────────────┐  │
│  │      Crypto Layer                           │  │
│  │  - Ed25519 Verification                     │  │
│  │  - Key Management                           │  │
│  │  - Signature Validation                     │  │
│  └────────────────┬────────────────────────────┘  │
│                   │                                │
│  ┌────────────────▼────────────────────────────┐  │
│  │      Storage Layer                          │  │
│  │  - .libreseed_modules/                      │  │
│  │  - Package Cache                            │  │
│  │  - Peer Cache                               │  │
│  └─────────────────────────────────────────────┘  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 4.3 Data Flow

#### Package Installation Flow

```
1. User: libreseed install lodash

2. Gateway receives request

3. DHT Lookup:
   Key: libreseed:lodash
   → Get infohash + publisher_pubkey + signature

4. Signature Verification:
   verify_ed25519(value, sig, publisher_pubkey)
   → If invalid, reject

5. Torrent Download:
   WebTorrent.add(infohash)
   → Connect to seeders (prioritize publisher)
   → Download to .libreseed_modules/lodash

6. Integrity Check:
   torrent.infoHash === expected_infohash
   → If mismatch, reject

7. Module Resolution Setup:
   - NODE_PATH=.libreseed_modules
   - OR require('libreseed/register')

8. Success:
   Package available for import
```

### 4.4 Technology Stack Summary

| Layer | Technology | Version | Rationale |
|-------|-----------|---------|-----------|
| **DHT** | @libp2p/kad-dht | Latest | NAT traversal, multi-transport, IPFS ecosystem |
| **Torrent** | webtorrent | Latest | Battle-tested, hybrid network, active maintenance |
| **Crypto** | @noble/ed25519 | Latest | Pure JS, audited, fast |
| **Gateway** | Express/Fastify | Latest | HTTP API server |
| **CLI** | Commander.js | Latest | Command-line interface |
| **Module Resolution** | Custom Hooks | Node.js 20+ | --loader flag, require() wrapper |

---

## 5. Implementation Roadmap

### Phase 1: Core Prototype (MVP)
**Timeline:** 4-6 weeks

- ✅ DHT client (libp2p/kad-dht) with custom validators
- ✅ Torrent client (WebTorrent) integration
- ✅ Ed25519 signature verification
- ✅ Basic gateway HTTP API (search, download)
- ✅ CLI tool (`libreseed install <package>`)
- ✅ NODE_PATH-based module resolution

**Deliverables:**
- Working prototype: install and use a package
- Basic publisher tool

### Phase 2: Publisher Tools
**Timeline:** 2-3 weeks

- ✅ Publisher CLI (`libreseed publish <path>`)
- ✅ Key generation and management
- ✅ Automated seeder daemon
- ✅ DHT publishing logic

### Phase 3: Module Resolution
**Timeline:** 3-4 weeks

- ✅ require() monkey-patch (CommonJS)
- ✅ --loader hook (ESM)
- ✅ Webpack plugin
- ✅ Vite plugin

### Phase 4: Production Readiness
**Timeline:** 4-6 weeks

- ✅ Gateway caching layer
- ✅ Bootstrap node federation
- ✅ Publisher discovery (bootstrap registry)
- ✅ Key revocation mechanism
- ✅ Seeder reputation system
- ✅ Performance optimization
- ✅ Comprehensive testing

### Phase 5: Ecosystem Growth
**Timeline:** Ongoing

- Community bootstrap nodes
- Mirror seeder network
- Package migration tools (npm → libreseed)
- Browser support (WebRTC gateway)

---

## 6. Open Questions for Stakeholders

1. **Governance:**
   - Who maintains the bootstrap publisher registry?
   - DAO? Foundation? Community GitHub repo?

2. **Compatibility:**
   - Should LibreSeed support npm fallback?
   - Should packages be dual-published (npm + libreseed)?

3. **Incentives:**
   - How to incentivize long-term seeding?
   - Should there be paid seeder services?

4. **Namespace:**
   - Allow only mirrored npm packages, or new packages?
   - Namespace collision resolution strategy?

5. **Security:**
   - Key revocation: who signs the revocation list?
   - What happens to packages from revoked keys?

---

## 7. Conclusion

**LibreSeed's architecture is fundamentally sound** and ready for implementation with the following key decisions:

✅ **ADOPT:**
- DHT: `@libp2p/kad-dht` (Kademlia)
- Torrent: `webtorrent`
- Crypto: `@noble/ed25519`
- Key scheme: `libreseed:<package-name>` (with scope support)

⚠️ **REQUIRES WORK:**
- Node.js module resolution: Multi-pronged approach (wrapper + hooks + plugins)
- Publisher discovery: Bootstrap registry + announcement protocol
- Bootstrap resilience: Federated nodes + DNS seeds + peer cache

🔴 **HIGH PRIORITY:**
- Implement key revocation mechanism (security)
- Design gateway to be stateless (decentralization)
- Create seeder incentive model (availability)

**Next Steps:**
1. Stakeholder decision on open questions (Section 6)
2. Begin Phase 1 implementation (MVP)
3. Recruit community for bootstrap node operators
4. Design publisher registry governance

---

## References

- LibreSeed Specification v1.1: `LIBRESEED-SPEC-v1.1.md`
- libp2p Documentation: https://docs.libp2p.io
- WebTorrent Documentation: https://webtorrent.io/docs
- Node.js Loader Hooks: https://nodejs.org/api/esm.html#loaders
- Mainline DHT Specification: BEP-0005

---

**Document Status:** ✅ COMPLETE  
**Review Status:** Pending Stakeholder Review  
**Last Updated:** 2025-01-27
