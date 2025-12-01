# Changelog

Tutte le modifiche rilevanti al progetto LibreSeed saranno documentate in questo file.

Il formato è basato su [Keep a Changelog](https://keepachangelog.com/it/1.0.0/),
e questo progetto aderisce al [Semantic Versioning](https://semver.org/lang/it/).

## [Non rilasciato]

### Aggiunto
- **Supporto firma duale pacchetti (Creator + Maintainer)**:
  - CLI `lbs add` ora visualizza sia Creator che Maintainer fingerprint
  - CLI `lbs list` ora mostra Maintainer fingerprint quando diverso dal Creator
  - Struct `PackageInfo` esteso con `MaintainerFingerprint` e `MaintainerManifestSignature`
  - Visualizzazione condizionale: Maintainer mostrato solo se diverso dal Creator
  - File modificati: `cmd/lbs/add.go` (linee 104-116), `cmd/lbs/list.go` (linee 24-28, 132-135)

## [0.3.0] - 2025-11-30

### Aggiunto
- Sistema completo di gestione pacchetti con firma crittografica
  - `pkg/crypto/keymanager.go` - Gestione ciclo di vita chiavi Ed25519 con fingerprinting
  - `pkg/daemon/package_manager.go` - Archiviazione metadata pacchetti con persistenza YAML
  - Integrazione KeyManager e PackageManager in daemon
- API HTTP per gestione pacchetti
  - `POST /packages/add` - Upload pacchetto con multipart form
  - `GET /packages/list` - Lista tutti i pacchetti
  - `DELETE /packages/remove` - Rimozione pacchetto per ID
- Comandi CLI per gestione pacchetti
  - `lbs add <file> <name> <version> [description]` - Aggiunge pacchetto al daemon
  - `lbs list` - Lista tutti i pacchetti con formato tabulare e dettagli

### Corretto
- **Bug critico DHT announcement persistence**:
  - Pacchetti non venivano ri-annunciati al DHT dopo restart del daemon
  - Aggiunta sincronizzazione PackageManager → Announcer al startup in `pkg/daemon/daemon.go` (linee 152-175)
  - Conversione corretta da PackageID (hex string) a `metainfo.Hash` (20-byte array)
  - Logging dettagliato del processo di sync per debugging
  - Test suite completo aggiunto in `pkg/daemon/daemon_test.go` (505 linee, 6 test + benchmark)
  - Verificato end-to-end: pacchetti esistenti ora persistono announcements al DHT dopo restart
- **Bug critici routing API (Go 1.22+ compatibility)**:
  - Registrazione route HTTP aggiornata con sintassi method-aware (`"POST /path"` invece di `"/path"`)
  - Fix validazione dimensione file in `handlePackageAdd` (usato `header.Size` invece di `0`)
  - JSON struct tags in `cmd/lbs/list.go` aggiornati da snake_case a PascalCase per matching con API response
  - File modificati: `pkg/daemon/daemon.go` (linee 271-273), `pkg/daemon/handlers.go` (linea 93), `cmd/lbs/list.go` (linee 16-27)
- 8 errori di compilazione in `pkg/daemon/handlers.go`:
  - Parametri `w` e `r` invertiti in handler functions
  - Tipo `io.ReaderCloser` vs `io.Reader` in `addPackageHandler`
  - Type mismatch per `fingerprint` (string vs []byte)
  - Conversioni parametri URL e validazione richieste

### Modificato
- Versione CLI bumped da "dev" a "v0.3.0"
- Help message aggiornato con nuovi comandi `add` e `list`

### Note Tecniche
- Go 1.22+ ha modificato il comportamento di `http.ServeMux`: richiesta sintassi esplicita HTTP method nelle route
- Formato JSON response API usa PascalCase per field names (convenzione Go struct export)

## [0.2.0] - 2025-11-30

### Aggiunto
- Sistema di configurazione con supporto variabili d'ambiente (`LoadFromEnv()`)
  - Supporto per 10+ variabili di configurazione (`LIBRESEED_*`)
  - Campo `DHTBootstrapNodes` per nodi DHT pubblici predefiniti
  - Validazione migliorata per configurazione DHT
  - Percorsi predefiniti conformi a XDG Base Directory Specification
- Implementazione completa daemon `lbsd` con integrazione DHT
  - Server HTTP per API di gestione
  - Integrazione con BitTorrent DHT per scoperta peer
  - Gestione stato daemon e statistiche runtime
  - Supporto per seeding e annunci DHT
- Comandi CLI `lbs` per controllo daemon
  - `lbs start` - Avvio daemon in background
  - `lbs stop` - Arresto graceful del daemon
  - `lbs status` - Verifica stato daemon
  - `lbs stats` - Statistiche runtime del daemon
  - `lbs restart` - Riavvio completo del daemon

### Corretto
- **Bug #5**: Comandi client (`lbs status`, `lbs stats`) riportavano erroneamente "STOPPED" con daemon attivo
  - Formato PID file migliorato a `PID:ADDRESS` per scoperta corretta dell'indirizzo
  - Corretta race condition scrivendo PID file prima dell'avvio server HTTP
  - Aggiunta funzione `getDaemonAddr()` per scoperta affidabile dell'indirizzo
  - Mantenuta retrocompatibilità con vecchio formato PID file
  - File modificati: `cmd/lbsd/main.go`, `cmd/lbs/start.go`, `cmd/lbs/status.go`, `cmd/lbs/stats.go`

### Modificato
- Specifica 002 (CLI rename & install) aggiornata con apprendimenti della sessione
  - Aggiunti requisiti di installazione transazionale (FR-026/027/028)
  - Aggiunti requisiti di gestione errori (FR-037-040)
  - Espanso criterio di successo per rilevamento avvio e rollback (SC-011/012)
  - Aggiunta sezione strategia di testing completa
  - Documentati apprendimenti di processo e razionale decisioni

### Automazione
- **Makefile** completo con 20+ target per build, test, install
  - Build multipiattaforma con rilevamento automatico OS/architettura
  - Generazione e verifica checksum SHA-256
  - Target per linting, formatting, pulizia
  - Supporto per installazione system-wide e locale
- **Script di installazione** (`install.sh`) production-ready
  - Installazione transazionale con rollback automatico su fallimento
  - Backup automatico di binari esistenti prima dell'aggiornamento
  - Verifica prerequisiti (versione Go, Make, sha256sum)
  - Rilevamento piattaforma e validazione ambiente
  - Capacità di disinstallazione (`--uninstall`)
  - Gestione permessi e creazione directory di sistema
- **Script di test DHT** (`test-dht.sh`) per verifica integrazione

### Documentazione
- **DHT_INTEGRATION_COMPLETE.md**: Log completo sessione integrazione DHT
  - Documentate tutte le 8 fix di compilazione API DHT
  - Fix di validazione configurazione documentati
  - Risultati di verifica runtime inclusi
  - Decisioni tecniche e apprendimenti registrati
- **PROGRESS.md**: Tracciamento sviluppo per Fasi 1-3
  - Riassunti completamento fase
  - Metriche di sviluppo
  - Roadmap future features
- **manual-test-commands.md**: Guida riferimento test manuali
  - Procedure di test e troubleshooting
  - Comandi di verifica e diagnostica

### Note Tecniche
- Formato PID file: `PID:ADDRESS\n` (es. `2974845:127.0.0.1:9091\n`)
- Priorità scoperta indirizzo: PID file → env var `LIBRESEED_LISTEN_ADDR` → default `localhost:8080`
- Daemon forking: processo figlio gestito tramite `exec.Command` con detach completo
- Build system richiede Go 1.21+, GNU Make, sha256sum

## [0.1.0] - 2025-11-29

### Aggiunto
- Implementazione iniziale del tipo `PublicKey` in `pkg/crypto/keys.go`
  - Supporto per chiavi pubbliche Ed25519 (32 bytes)
  - Calcolo del fingerprint tramite SHA-256 (primi 8 bytes)
  - Verifica delle firme Ed25519 tramite metodo `Verify()`
  - Costruttore `NewPublicKey()` con validazione completa
  - Metodi helper `Bytes()` e `String()` per serializzazione e debug
- Implementazione del sistema di firma digitale in `pkg/crypto/signer.go`
  - Tipo `Signature` con metadati (algoritmo, firmatario, timestamp)
  - Funzione `Sign()` per creare firme Ed25519 a 64 bytes
  - Funzione `Verify()` per verificare l'autenticità delle firme
  - Funzione `SignatureFromBytes()` per deserializzazione firme
  - Metodi `Bytes()` e `String()` per gestione e debug delle firme
  - Costanti ed errori predefiniti per validazione robusta
- Documentazione completa del package `crypto` in italiano
- Validazione della lunghezza delle chiavi (esattamente 32 bytes)
- Validazione della lunghezza delle firme (esattamente 64 bytes)
- Gestione degli errori per chiavi/firme nil, vuote o con dimensione errata

### Note Tecniche
- Utilizzo di `crypto/ed25519` dalla standard library di Go
- Fingerprint generato come hex encoding dei primi 8 bytes di SHA-256
- Formato stringa: `algorithm:fingerprint` (es. `ed25519:a1b2c3d4e5f67890`)
- Nessuna dipendenza esterna per le operazioni crittografiche core

[0.2.0]: https://github.com/libreseed/libreseed/releases/tag/v0.2.0
[0.1.0]: https://github.com/libreseed/libreseed/releases/tag/v0.1.0
