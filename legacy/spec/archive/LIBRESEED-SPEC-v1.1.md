# 📘 LIBRESEED — SPECIFICA TECNICA COMPLETA

**Versione 1.1 — Gateway + Seeder**  
**Protocol Name:** `libreseed`  
**Data:** 2024-11-27

---

## 🟦 Indice

1. [Obiettivo del sistema](#1-obiettivo-del-sistema)
2. [Architettura generale](#2-architettura-generale)
3. [Identità e sicurezza](#3-identità-e-sicurezza)
4. [DHT Manifest Format](#4-dht-manifest-format)
5. [DHT Keys & Lookup](#5-dht-keys--lookup)
6. [Gateway NPM](#6-gateway-npm)
7. [Seeder](#7-seeder)
8. [Torrent Package Structure](#8-torrent-package-structure)
9. [Algoritmi principali](#9-algoritmi-principali)
10. [Error Handling](#10-error-handling)
11. [Esempi](#11-esempi)
12. [Glossario](#12-glossario)
13. [Punti aperti e decisioni da prendere](#13-punti-aperti-e-decisioni-da-prendere)

---

## 1. 🎯 Obiettivo del sistema

Costruire un registry decentralizzato per pacchetti software chiamato **LibreSeed** con:

- ✅ Nessun server centrale
- ✅ Nessun costo
- ✅ Distribuzione file via BitTorrent
- ✅ Metadati, versioni e name resolution via DHT (Kademlia)
- ✅ Integrità tramite firme digitali
- ✅ Installazione tramite un solo pacchetto NPM gateway
- ✅ Persistenza garantita da una rete di seeder decentralizzati

---

## 2. 🏗️ Architettura generale

```
Publisher → (manifest + torrent) → DHT + Torrent network
Seeder ← downloads all packages & manifests → seeds
Gateway NPM → resolves requests → downloads via torrent → validates → installs
```

### Componenti:

1. **Manifest Layer (DHT)**  
   Versioni, infohash, metadata, firma digitale, latest pointers.

2. **Data Layer (Torrent)**  
   Il pacchetto reale (file software).

3. **Gateway Layer (NPM)**  
   Installazione, validazione e risoluzione.

4. **Seeder Layer**  
   Mantenimento disponibilità, raccolta di tutti i pacchetti validi, re-seeding.

---

## 3. 🔐 Identità e sicurezza

### 3.1 Keypair del Publisher

Ogni publisher possiede:

- `pubkey` (base64, pubblica)
- `privkey` (segreta, mai condivisa)

### 3.2 Firma dei Manifest

Ogni manifest è firmato:

```javascript
signature = sign(privkey, canonicalJSON(manifest))
```

### 3.3 Validazione nel Gateway

```javascript
verify(pubkey, signature, canonicalJSON(manifest)) → true/false
```

### 3.4 Invarianti di sicurezza

- ❌ Nessun record non firmato è accettato
- ❌ Nessun record con firma invalida è accettato
- ❌ Nessuno può pubblicare versioni fasulle senza la privkey
- ✅ `@latest` deve essere firmato e coerente con il manifest puntato

### 3.5 Revoca chiavi

**Policy**: Non esiste meccanismo di revoca.

**Rationale**:
- Sistema completamente decentralizzato
- Nessuna autorità centrale per gestire revoche
- Responsabilità del publisher proteggere la propria chiave privata
- In caso di compromissione: publisher deve pubblicare nuovo pacchetto con nuovo nome e nuova keypair

---

## 4. 📄 DHT Manifest Format

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "abcdef0123456789...",
  "pubkey": "base64-publisher-key",
  "signature": "base64-signature",
  "timestamp": 1733123456,
  "metadata": {
    "description": "Optional metadata",
    "homepage": "Optional URL",
    "dependencies": {}
  }
}
```

### Campi obbligatori:

- `protocol`: DEVE essere `"libreseed-v1"`
- `name`: Nome del pacchetto
- `version`: Versione semver
- `infohash`: Hash del torrent
- `pubkey`: Chiave pubblica del publisher
- `signature`: Firma digitale del manifest
- `timestamp`: Unix timestamp della pubblicazione

### Campi opzionali:

- `metadata`: Metadati addizionali (descrizione, homepage, dipendenze, etc.)

---

## 5. 🔑 DHT Keys & Lookup

### 5.1 Chiavi DHT

#### Version-specific manifest:
```
sha256(name + "@" + version)
```

**Esempio**: `sha256("mypackage@1.4.0")`

#### Latest pointer:
```
sha256(name + "@latest")
```

**Esempio**: `sha256("mypackage@latest")`

**Contenuto**:
```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "pointer": "sha256(mypackage@1.4.0)",
  "pubkey": "base64-publisher-key",
  "signature": "base64-signature",
  "timestamp": 1733123456
}
```

#### Publisher Announce:
```
sha256("libreseed:announce:" + pubkey)
```

**Esempio**: `sha256("libreseed:announce:ABC123...")`

### 5.2 Announce Format

**Ogni publisher pubblica il proprio announce** contenente tutti i suoi pacchetti:

```json
{
  "protocol": "libreseed-v1",
  "announceVersion": "1.0.0",
  "pubkey": "base64-publisher-key",
  "timestamp": 1733123456,
  "packages": [
    {
      "name": "mypackage",
      "versions": [
        {
          "version": "1.4.0",
          "pointer": "sha256(mypackage@1.4.0)",
          "timestamp": 1733120000
        },
        {
          "version": "1.3.0",
          "pointer": "sha256(mypackage@1.3.0)",
          "timestamp": 1733110000
        }
      ],
      "latest": "1.4.0"
    },
    {
      "name": "toolkit",
      "versions": [
        {
          "version": "2.0.0",
          "pointer": "sha256(toolkit@2.0.0)",
          "timestamp": 1733100000
        }
      ],
      "latest": "2.0.0"
    }
  ],
  "signature": "base64-signature"
}
```

### 5.3 Announce Protocol

**Chi lo pubblica**: Ogni publisher pubblica il proprio announce.

**Quando viene aggiornato**: Durante ogni `libreseed publish`.

**Chiave DHT**: `sha256("libreseed:announce:" + pubkey)`

**Versionamento**: Il campo `announceVersion` permette futuri cambiamenti di formato.

**Firma**: L'intero announce è firmato dal publisher.

### 5.4 Discovery dei Publisher

**Problema aperto**: Come scoprire tutti i publisher esistenti?

**Opzioni possibili** (da discutere in sezione 13):

1. **Lista hardcoded**: Gateway contiene lista di publisher "noti"
2. **Discovery DHT**: Chiave speciale `sha256("libreseed:publishers")` con lista aggregata
3. **Web-of-trust**: Ogni publisher referenzia altri publisher
4. **Bootstrap nodes**: Nodi speciali che mantengono lista completa
5. **Hybrid**: Combinazione di approcci

---

## 6. 🟩 Gateway NPM

Pacchetto unico pubblicato in npm:

**Nome**: `libreseed-gateway` (o `gateway-libreseed`)

### 6.1 Funzioni principali

- ✅ Risoluzione semver decentralizzata
- ✅ Lookup DHT → manifest → validazione firme
- ✅ Download torrent file
- ✅ Verifica integrità torrent
- ✅ Estrazione in directory dedicata (fuori da `node_modules`)
- ✅ Esposizione path/bin per essere consumati da altri pacchetti
- ✅ Caching locale con refresh automatico
- ✅ Gestione errore con retry su nodi multipli

### 6.2 Configurazione in package.json

```json
{
  "name": "my-app",
  "dependencies": {
    "a-soft": "npm:libreseed-gateway",
    "b-soft": "npm:libreseed-gateway",
    "c-soft": "npm:libreseed-gateway"
  },
  "libreseedConfig": {
    "a-soft": {
      "spec": "mypackage@^1.2.0",
      "publisher": "ABC123..."
    },
    "b-soft": {
      "spec": "toolkit@latest",
      "publisher": "XYZ789..."
    },
    "c-soft": {
      "spec": "anotherpkg@2.0.0",
      "publisher": "DEF456..."
    }
  }
}
```

**Nota**: Il campo `publisher` (pubkey) è **obbligatorio** per identificare l'announce corretto.

### 6.3 Lifecycle

#### Step 1 — Resolve spec

Input possibile:

- `"mypackage"` → implica `@latest`
- `"mypackage@^2.0.0"` → range semver
- `"mypackage@2.3.0"` → versione esatta
- `"torrent:infohash"` → download diretto (bypass DHT)
- `"magnet:?..."` → magnet link diretto

**Flusso risoluzione name-based**:

1. Lookup `sha256(publisher + ":announce:" + pubkey)`
2. Validazione firma announce
3. Estrazione lista versioni per `name`
4. Se richiesto `@latest`: usa campo `latest` dell'announce
5. Se range semver: filtra versioni compatibili e prendi la maggiore
6. Lookup manifest: `sha256(name + "@" + version)`
7. Validazione firma manifest

#### Step 2 — Download torrent

1. Avvia client torrent con `infohash` dal manifest
2. Verifica hash dei pezzi durante download
3. **Se fallisce**: Applica logica di retry (vedi sezione 10.2)

#### Step 3 — Install

**Posizione installazione**:

```
.libreseed_modules/<alias>/
```

**Rationale**:
- ✅ Separazione da `node_modules` per compatibilità con pnpm/yarn
- ✅ Evita conflitti con package manager nativi
- ✅ Permette gestione indipendente del ciclo di vita

**Struttura**:
```
.libreseed_modules/
  a-soft/
    manifest.json
    dist/
    src/
  b-soft/
    manifest.json
    dist/
```

### 6.4 Module Resolution

**Problema**: Come permettere a Node.js di risolvere `require('a-soft')` o `import 'a-soft'`?

**Opzioni da valutare** (vedi sezione 13):

1. **Symlink automatici**:
   ```
   node_modules/a-soft → ../.libreseed_modules/a-soft
   ```
   - ✅ Compatibile con tutti i bundler
   - ❌ Richiede permessi per creare symlink (problemi su Windows)

2. **NODE_PATH environment variable**:
   ```bash
   NODE_PATH=.libreseed_modules node app.js
   ```
   - ✅ Semplice
   - ❌ Non funziona con bundler (webpack, vite)

3. **Wrapper API**:
   ```javascript
   const libreseed = require('libreseed-gateway');
   const aSoft = libreseed.require('a-soft');
   ```
   - ✅ Controllo completo
   - ❌ Non supporta `import` ES6 nativo
   - ❌ Richiede modifica del codice utente

4. **Custom loader (Node.js --loader)**:
   ```bash
   node --loader libreseed-loader app.js
   ```
   - ✅ Supporta `import` ES6
   - ✅ Trasparente per l'utente
   - ❌ Richiede flag runtime
   - ❌ API sperimentale in Node.js

5. **Package.json exports mapping**:
   ```json
   {
     "exports": {
       "./a-soft": "./.libreseed_modules/a-soft/dist/index.js"
     }
   }
   ```
   - ✅ Standard Node.js
   - ❌ Richiede generazione dinamica del `package.json` del gateway

### 6.5 TypeScript Typings Support

**Problema**: Come esporre i typings (`.d.ts`) dei pacchetti LibreSeed?

**Opzioni da valutare** (vedi sezione 13):

1. **Symlink `@types/` directory**:
   ```
   node_modules/@types/a-soft → ../.libreseed_modules/a-soft/dist/index.d.ts
   ```

2. **tsconfig.json paths**:
   ```json
   {
     "compilerOptions": {
       "paths": {
         "a-soft": [".libreseed_modules/a-soft/dist"]
       }
     }
   }
   ```
   - ✅ Standard TypeScript
   - ❌ Richiede configurazione manuale o auto-generazione

3. **Typings bundle nel gateway**:
   Gateway contiene stub typings che re-esportano i veri typings.

### 6.6 Cache Policy

**Comportamento**:

- ✅ **Il gateway ri-controlla DHT ad ogni risoluzione**
- ✅ Nessun caching di `@latest` (sempre fetch fresco)
- ✅ Versioni esatte (`1.2.3`) possono essere cachate localmente dopo download
- ✅ Cache invalidation manuale con flag: `--libreseed-clear-cache`

**Directory cache**:
```
~/.libreseed/cache/
  torrents/
  manifests/
```

---

## 7. 🟦 Seeder

Software dedicato, Dockerizzabile.

### 7.1 Obiettivi

- ✅ Garantire persistenza pacchetti
- ✅ Seed costante dei torrent
- ✅ Mirroring decentralizzato dei package reali
- ✅ Garbage collection torrent orfani
- ✅ Priorità per pacchetti propri dello sviluppatore
- ✅ Enforcement di limiti disco
- ✅ **Re-put DHT periodico per mantenere record alive**

### 7.2 Config YAML

```yaml
protocolIDs:
  - "libreseed-v1"

# Publisher di cui seedare tutti i pacchetti
trackedPublishers:
  - "<base64-pubkey-1>"
  - "<base64-pubkey-2>"

# Pacchetti propri (priorità massima)
ownPackages:
  - "mypkg"
  - "toolkit"

# Limiti risorse
maxDiskGB: 300
minSeedersThreshold: 2

# Percorsi
storagePath: "/data/libreseed"
cachePath: "/cache/libreseed"

# Timing
announcePollSec: 600          # 10 minuti
refreshIntervalSec: 43200     # 12 ore (re-put DHT)
integrityCheckSec: 3600       # 1 ora

# Policy
allowUnknownPublishers: false  # Se true, seeda anche publisher non in trackedPublishers
```

### 7.3 Startup Sequence

1. Carica configurazione
2. Per ogni `pubkey` in `trackedPublishers`:
   - Recupera announce: `sha256("libreseed:announce:" + pubkey)`
   - Valida firma announce
3. Per ogni pacchetto nell'announce:
   - Recupera manifest → verifica firma
   - Verifica signature `@latest` pointer
   - Se torrent non presente localmente: scarica
   - Verifica hash dei pezzi
   - Avvia seeding
4. Rimuove torrent non più presenti nell'announce
5. Pulisce file corrotti
6. Pubblica `seederStatus` (vedi 7.6)

### 7.4 Permanent Loop

**Ogni `announcePollSec` (10 min)**:
- Refresh announce per ogni publisher tracciato
- Aggiunta/rimozione torrent in base a delta
- Aggiornamento seed-list

**Ogni `refreshIntervalSec` (12 ore)**:
- **Re-put di tutti i manifest nel DHT** (mantiene record alive)
- Re-put del proprio `seederStatus`

**Ogni `integrityCheckSec` (1 ora)**:
- Verifica integrità file torrent (hash check)
- LRU eviction se necessario
- Update disco utilizzato
- Seed-balance (mantieni `minSeedersThreshold`)

### 7.5 Seeding Policies

#### Priorità assoluta (sempre seedati):
1. Pacchetti in `ownPackages`
2. Pacchetti con `seeders < minSeedersThreshold`

#### Medio livello:
3. Pacchetti nuovi (timestamp recente)
4. Pacchetti popolari (alto download count, se disponibile)

#### Eviction LRU (quando spazio insufficiente):
5. Pacchetti con più seeders
6. Non appartenenti a `ownPackages`
7. Usati meno recentemente

#### Matrice di priorità (in caso di conflitto `minSeedersThreshold` vs `maxDiskGB`):

| Situazione | Policy |
|-----------|--------|
| Spazio disponibile + seeders < min | ✅ Seeda |
| Spazio disponibile + seeders >= min | ✅ Seeda se nuovo/popolare |
| Spazio insufficiente + ownPackages | ✅ Seeda sempre (evict altri) |
| Spazio insufficiente + seeders < min | ✅ Seeda (evict LRU non-critical) |
| Spazio insufficiente + seeders >= min | ❌ Non seeda o evict |

### 7.6 Seeder Status Publication

Ogni seeder pubblica il proprio status nel DHT:

**Chiave**: `sha256("libreseed:seeder:" + seeder_id)`

**Formato**:
```json
{
  "protocol": "libreseed-v1",
  "seederID": "unique-seeder-id",
  "timestamp": 1733123456,
  "seedingPackages": [
    {
      "name": "mypackage",
      "version": "1.4.0",
      "infohash": "abc123..."
    }
  ],
  "diskUsage": {
    "used": 250,
    "max": 300,
    "unit": "GB"
  },
  "uptime": 86400,
  "signature": "seeder-signature"
}
```

**Utilizzo**:
- Permette discovery di seeder attivi
- Monitoring decentralizzato della rete
- Coordinamento implicito per evitare over-seeding

---

## 8. 📦 Torrent Package Structure

Dentro il torrent:

```
/
  manifest.json        # Manifest identico a quello nel DHT
  dist/                # Binari, JS bundle, libs
  src/                 # Opzionale: codice sorgente
  docs/                # Opzionale: documentazione
  package.json         # Opzionale: metadata NPM-like
```

### Requisiti:

- ✅ **`manifest.json` DEVE essere identico bit-per-bit al manifest DHT**
  - Rationale: Permette validazione offline e recovery
  - Il gateway DEVE verificare che `sha256(manifest.json) == sha256(manifest_dht)`

- ✅ Tutti i file del pacchetto devono essere inclusi
- ✅ Struttura directory deve seguire convenzioni (es. `dist/` per build output)

---

## 9. 🔍 Algoritmi principali

### 9.1 Resolve Latest

```javascript
function resolveLatest(name, pubkey) {
  // 1. Get announce
  const announceKey = sha256("libreseed:announce:" + pubkey);
  const announce = DHT.get(announceKey);
  validateSignature(announce);

  // 2. Find package in announce
  const pkg = announce.packages.find(p => p.name === name);
  if (!pkg) throw new Error("Package not found in announce");

  // 3. Get latest version
  const latestVersion = pkg.latest;
  const manifestKey = sha256(name + "@" + latestVersion);
  const manifest = DHT.get(manifestKey);
  validateSignature(manifest);

  return manifest;
}
```

### 9.2 Resolve Semver

```javascript
function resolveSemver(name, range, pubkey) {
  // 1. Get announce
  const announceKey = sha256("libreseed:announce:" + pubkey);
  const announce = DHT.get(announceKey);
  validateSignature(announce);

  // 2. Find package in announce
  const pkg = announce.packages.find(p => p.name === name);
  if (!pkg) throw new Error("Package not found in announce");

  // 3. Filter versions by semver range
  const validVersions = pkg.versions.filter(v => 
    semver.satisfies(v.version, range)
  );

  if (validVersions.length === 0) {
    throw new Error("No version satisfies range: " + range);
  }

  // 4. Select highest version
  const selectedVersion = semver.maxSatisfying(
    validVersions.map(v => v.version), 
    range
  );

  // 5. Get manifest
  const manifestKey = sha256(name + "@" + selectedVersion);
  const manifest = DHT.get(manifestKey);
  validateSignature(manifest);

  return manifest;
}
```

**Nota**: Questo algoritmo **non richiede enumerazione completa del DHT**, poiché tutte le versioni sono già elencate nell'announce del publisher.

### 9.3 Seeder Startup

```javascript
function seederStartup(config) {
  for (const pubkey of config.trackedPublishers) {
    // 1. Get announce
    const announceKey = sha256("libreseed:announce:" + pubkey);
    const announce = DHT.get(announceKey);
    validateSignature(announce);

    // 2. Process each package
    for (const pkg of announce.packages) {
      for (const versionInfo of pkg.versions) {
        // 3. Get manifest
        const manifest = DHT.get(versionInfo.pointer);
        validateSignature(manifest);

        // 4. Download if not exists
        if (!localExists(manifest.infohash)) {
          downloadTorrent(manifest.infohash);
        }

        // 5. Verify integrity
        verifyTorrentIntegrity(manifest.infohash);

        // 6. Start seeding
        seedTorrent(manifest.infohash);
      }
    }
  }

  // 7. Cleanup
  removeOrphanedTorrents();
  publishSeederStatus();
}
```

### 9.4 DHT Re-put (Seeder)

```javascript
function dhtRePut() {
  // Re-publish all manifests to keep them alive in DHT
  for (const manifest of localManifests) {
    const key = sha256(manifest.name + "@" + manifest.version);
    DHT.put(key, manifest);
  }

  // Re-publish seeder status
  publishSeederStatus();
}

// Called every refreshIntervalSec (12 hours)
setInterval(dhtRePut, config.refreshIntervalSec * 1000);
```

**Rationale**: I record DHT hanno TTL limitato (tipicamente 24-48 ore in Kademlia). Il re-put periodico garantisce che i manifest non spariscano dal DHT.

---

## 10. 🚨 Error Handling

### 10.1 Gateway Error Cases

| Errore | Azione |
|--------|--------|
| Manifest firma invalida | Fallback a versione precedente (se semver range), altrimenti abort |
| Manifest non esistente | Errore esplicito all'utente con dettagli |
| Announce non trovato | Errore esplicito: publisher sconosciuto o offline |
| Torrent non scaricabile | Applica logica retry (vedi 10.2) |
| Hash mismatch | Marcato come corrotto, escluso da retry, passa a versione precedente |
| Timeout DHT | Retry con nodi diversi + fallback multipath |
| Publisher pubkey mancante | Errore: configurazione incompleta |

### 10.2 Retry Logic con Blacklist

**Policy**: Dopo **10 tentativi falliti**, una versione viene blacklisted localmente.

```javascript
const blacklist = new Map(); // version -> failCount

function downloadWithRetry(manifest, maxRetries = 10) {
  const versionKey = manifest.name + "@" + manifest.version;
  
  if (blacklist.get(versionKey) >= maxRetries) {
    throw new Error("Version blacklisted after " + maxRetries + " failures");
  }

  try {
    downloadTorrent(manifest.infohash);
    // Success: reset counter
    blacklist.delete(versionKey);
  } catch (error) {
    // Increment failure counter
    const count = (blacklist.get(versionKey) || 0) + 1;
    blacklist.set(versionKey, count);

    if (count >= maxRetries) {
      throw new Error("Version blacklisted: " + versionKey);
    }

    // Exponential backoff
    const delay = Math.min(1000 * Math.pow(2, count), 60000);
    await sleep(delay);
    
    // Retry
    return downloadWithRetry(manifest, maxRetries);
  }
}
```

**Fault tolerance**:
- ✅ Retry automatico con backoff esponenziale
- ✅ Blacklist locale per evitare loop infiniti
- ✅ Fallback a versione precedente (se semver range)
- ✅ Cache failure reset su successo

### 10.3 Seeder Error Cases

| Errore | Azione |
|--------|--------|
| Manifest invalido | Rimozione pacchetto dalla lista seed |
| Torrent corrotto | Delete locale + re-download |
| Spazio insufficiente | Eviction LRU (vedi policy 7.5) |
| DHT announce irraggiungibile | Continua seeding locale, retry announce poll |
| Signature verification failure | Skip pacchetto, log warning |

---

## 11. 📘 Esempi

### 11.1 package.json con 3 pacchetti LibreSeed

```json
{
  "name": "my-app",
  "version": "1.0.0",
  "dependencies": {
    "a-soft": "npm:libreseed-gateway",
    "b-soft": "npm:libreseed-gateway",
    "c-soft": "npm:libreseed-gateway"
  },
  "libreseedConfig": {
    "a-soft": {
      "spec": "mypackage@^1.2.0",
      "publisher": "ABC123pubkey"
    },
    "b-soft": {
      "spec": "toolkit@latest",
      "publisher": "XYZ789pubkey"
    },
    "c-soft": {
      "spec": "anotherpkg@2.0.0",
      "publisher": "DEF456pubkey"
    }
  }
}
```

### 11.2 Publish Workflow (Publisher)

```bash
# 1. Crea manifest
libreseed manifest create mypackage 1.4.0 --dist ./dist

# 2. Firma manifest
libreseed manifest sign --key ~/.libreseed/privkey.pem

# 3. Crea torrent
libreseed torrent create ./dist --manifest manifest.json

# 4. Pubblica su DHT
libreseed publish manifest.json

# 5. Aggiorna announce
libreseed announce update --add mypackage@1.4.0

# 6. (Opzionale) Seed torrent
libreseed seed manifest.json
```

### 11.3 Install Workflow (User)

```bash
# 1. Install gateway
npm install

# 2. Gateway risolve e installa automaticamente
# (lifecycle hook in libreseed-gateway)

# 3. Usa pacchetto
node app.js
# import aSoft from 'a-soft';  (se module resolution configurato)
```

---

## 12. 📚 Glossario

| Termine | Significato |
|---------|------------|
| **LibreSeed** | Nome del protocollo e del sistema di registry decentralizzato |
| **Manifest DHT** | Documento firmato con version info + infohash torrent pubblicato nel DHT |
| **Seeder** | Nodo persistente che scarica e seed-a i pacchetti LibreSeed |
| **Gateway** | Pacchetto NPM che installa pacchetti LibreSeed (`libreseed-gateway`) |
| **Announce** | Lista dei pacchetti validi pubblicata da ogni publisher |
| **Latest** | Record speciale che punta alla versione più recente di un pacchetto |
| **Semver** | Versionamento semantico (Semantic Versioning) |
| **DHT** | Distributed Hash Table (Kademlia) per metadati |
| **Infohash** | Hash SHA-1 del torrent file (identifica univocamente un torrent) |
| **Publisher** | Entità che possiede keypair e pubblica pacchetti |
| **Blacklist** | Lista locale di versioni con troppi download falliti |
| **Re-put** | Ri-pubblicazione periodica di record DHT per mantenerli alive |

---

## 13. 🔴 Punti aperti e decisioni da prendere

### 13.1 🚨 CRITICO: Enumerazione versioni semver

**Problema risolto**: ✅ Grazie agli announce versionati, non è più necessario enumerare il DHT.

**Algoritmo**:
1. Lookup announce del publisher
2. Estrai lista completa versioni dal campo `packages[].versions`
3. Filtra con semver
4. Seleziona versione maggiore

**Vantaggio**: O(1) lookup invece di O(n) scan DHT.

---

### 13.2 🚨 CRITICO: Publisher Discovery

**Problema**: Come scopre il gateway quali publisher esistono?

**Stato**: ⚠️ **DA DECIDERE**

**Opzioni**:

#### A) Hardcoded bootstrap list
```javascript
const KNOWN_PUBLISHERS = [
  "ABC123...",
  "XYZ789...",
  // ...
];
```
- ✅ Semplice
- ✅ Deterministico
- ❌ Centralizzato (lista nel codice gateway)
- ❌ Richiede update gateway per nuovi publisher

#### B) DHT aggregated list
```
sha256("libreseed:publishers") → {
  "publishers": ["ABC...", "XYZ..."],
  "signature": "???"  // Chi firma?
}
```
- ✅ Decentralizzato
- ❌ Chi ha autorità di scrivere questa lista?
- ❌ Rischio spam/pollution

#### C) Web-of-trust
Ogni publisher referenzia altri publisher nel suo announce:
```json
{
  "trustedPublishers": ["ABC...", "XYZ..."]
}
```
- ✅ Decentralizzato
- ✅ Social graph naturale
- ❌ Complessità discovery (BFS su grafo)
- ❌ Rischio isole disconnesse

#### D) Bootstrap nodes
Nodi speciali (non DHT) che mantengono registry di publisher:
```
GET https://bootstrap.libreseed.org/publishers
```
- ✅ Semplice
- ❌ Centralizzato (bootstrap nodes)
- ❌ Single point of failure

#### E) Configurazione utente esplicita
```json
"libreseedConfig": {
  "knownPublishers": ["ABC...", "XYZ..."]
}
```
- ✅ Massimo controllo utente
- ✅ Nessuna dipendenza esterna
- ❌ Onere su utente finale

**Decisione richiesta**: Quale approccio (o combinazione) usare?

---

### 13.3 🟡 Module Resolution Strategy

**Problema**: Come far funzionare `import 'a-soft'` con pacchetti in `.libreseed_modules/`?

**Stato**: ⚠️ **DA DECIDERE**

**Opzioni riepilogo**:

| Approccio | Pro | Contro |
|-----------|-----|--------|
| **Symlink** | Compatibilità universale | Problemi permessi Windows |
| **NODE_PATH** | Semplice | Non funziona con bundler |
| **Wrapper API** | Controllo totale | Non supporta `import` ES6 |
| **Custom loader** | Supporta `import` | API sperimentale, flag runtime |
| **package.json exports** | Standard Node.js | Richiede generazione dinamica |

**Decisione richiesta**: Quale strategia implementare?

**Proposta ibrida** (da validare):
1. **Default**: Symlink (funziona out-of-the-box)
2. **Fallback**: Se symlink fallisce, usa wrapper API
3. **Opzionale**: Custom loader per advanced users

---

### 13.4 🟡 TypeScript Typings

**Problema**: Come esporre `.d.ts` per TypeScript IntelliSense?

**Stato**: ⚠️ **DA DECIDERE**

**Opzioni**:

#### A) Symlink `@types/`
```
node_modules/@types/a-soft → ../.libreseed_modules/a-soft/dist/index.d.ts
```

#### B) tsconfig.json auto-generazione
Gateway genera automaticamente:
```json
{
  "compilerOptions": {
    "paths": {
      "a-soft": [".libreseed_modules/a-soft/dist"]
    }
  }
}
```

#### C) Typings bundle
Gateway contiene stub `.d.ts` che re-esporta:
```typescript
// node_modules/libreseed-gateway/typings/a-soft.d.ts
export * from '../../.libreseed_modules/a-soft/dist';
```

**Decisione richiesta**: Quale approccio garantisce migliore DX?

---

### 13.5 🟢 Seeder ID Generation

**Problema**: Come genera un seeder il suo `seederID` univoco?

**Proposta**:
```javascript
seederID = sha256(pubkey + random_nonce)
```

**Opzioni alternative**:
- UUID v4
- Hash dell'IP + timestamp
- Keypair dedicata del seeder

**Decisione richiesta**: Confermare approccio.

---

### 13.6 🟢 Announce Size Limit

**Problema**: Se un publisher ha 10.000 pacchetti, l'announce diventa troppo grande.

**Proposta**: Limite pragmatico di **1000 pacchetti per announce**.

**Gestione overflow**:
- Announce multipli numerati: `libreseed:announce:pubkey:0`, `libreseed:announce:pubkey:1`, etc.
- Field `announceIndex` e `totalAnnounces` nel formato

**Decisione richiesta**: Confermare limite e strategia paginazione.

---

### 13.7 🟢 DHT Implementation

**Problema**: Quale implementazione DHT usare?

**Opzioni**:

| Libreria | Linguaggio | Pro | Contro |
|----------|-----------|-----|--------|
| **webtorrent-dht** | JavaScript | Compatibilità BitTorrent DHT | Performance |
| **libp2p-kad-dht** | JavaScript | Moderno, estensibile | Meno mature |
| **go-libp2p-kad-dht** | Go | Performance, produzione | Integrazione con Node.js |
| **mainline-dht** | Rust | Performance | Binding Node.js |

**Decisione richiesta**: Scelta implementazione per gateway e seeder.

---

### 13.8 🟢 Torrent Client

**Problema**: Quale client torrent usare nel gateway e seeder?

**Opzioni**:

| Client | Pro | Contro |
|--------|-----|--------|
| **WebTorrent** | Pure JS, browser-compatible | Performance |
| **libtorrent (via binding)** | Performance, maturo | Dipendenza nativa |
| **Transmission (daemon)** | Robusto, CLI-friendly | Deploy complexity |

**Decisione richiesta**: Client per gateway vs seeder (possono essere diversi).

---

## 14. 🎯 Next Steps

### Fase 1: Validazione Spec
- [ ] Review tecnica da team multi-agente
- [ ] Risoluzione punti aperti (sezione 13)
- [ ] Diagrammi di sequenza UML
- [ ] Threat modeling e security review

### Fase 2: Prototipo
- [ ] Gateway MVP (solo versioni esatte, no semver)
- [ ] Seeder MVP (singolo publisher)
- [ ] DHT testnet
- [ ] CLI publisher tool

### Fase 3: Production
- [ ] Semver completo
- [ ] Multi-publisher support
- [ ] Monitoring e metrics
- [ ] Documentazione utente finale

---

## 15. 📝 Changelog

### v1.1 (2024-11-27)
- ✅ Rinominato protocollo da "p2p-registry" a "libreseed"
- ✅ Announce versionati per publisher (un announce per publisher)
- ✅ Retry logic con blacklist (n=10)
- ✅ Politica "nessuna revoca chiavi"
- ✅ Manifest nel torrent identico a DHT
- ✅ Directory installazione `.libreseed_modules/` (non `node_modules`)
- ✅ Cache policy: sempre refresh DHT
- ✅ Re-put DHT periodico nei seeder (ogni 12h)
- ✅ Matrice priorità seeder con gestione conflitti disco
- ✅ Sezione "Punti aperti" con decisioni da prendere
- ✅ Algoritmo semver senza enumerazione DHT (usa announce)

### v1.0 (2024-11-26)
- Versione iniziale della specifica

---

**Fine della specifica v1.1**

🚀 Pronti per review tecnica multi-agente!
