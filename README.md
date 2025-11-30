# LibreSeed

**Sistema decentralizzato di distribuzione software tramite DHT BitTorrent**

LibreSeed è una soluzione moderna per la distribuzione peer-to-peer di pacchetti software, utilizzando la DHT (Distributed Hash Table) di BitTorrent per garantire disponibilità, resilienza e decentralizzazione.

## Indice

- [Caratteristiche](#caratteristiche)
- [Quick Start](#quick-start)
- [Guida per Sviluppatori](#guida-per-sviluppatori)
- [Architettura](#architettura)
- [Licenza](#licenza)

---

## Caratteristiche

✅ **Decentralizzato** — Nessun server centrale, discovery tramite DHT BitTorrent  
✅ **Resiliente** — Distribuzione peer-to-peer con ridondanza automatica  
✅ **CLI moderna** — Interfaccia a riga di comando intuitiva per gestione daemon  
✅ **Daemon robusto** — Servizio in background con graceful shutdown  
✅ **Monitoraggio** — Statistiche in tempo reale e stato del sistema  
✅ **Automazione completa** — Makefile con 20+ target per build, test, release  

---

## Quick Start

### Prerequisiti

- **Go** 1.21 o superiore
- **Make** (per build automation)
- **Git** (per clonare il repository)

### Installazione Rapida

```bash
# Clona il repository
git clone https://github.com/fulgidus/libreseed.git
cd libreseed

# Installa automaticamente
./install.sh
```

Lo script `install.sh` esegue:
- Verifica dei prerequisiti (Go, Make, sha256sum)
- Build dei binari (`lbs`, `lbsd`)
- Generazione e verifica dei checksum
- Installazione in `/usr/local/bin` (richiede sudo)
- Creazione delle directory dati in `~/.local/share/libreseed`

### Utilizzo Base

```bash
# Avvia il daemon
lbs start

# Verifica lo stato
lbs status

# Visualizza statistiche
lbs stats

# Ferma il daemon
lbs stop

# Riavvia il daemon
lbs restart

# Mostra versione
lbs version
```

### Struttura Directory

```
~/.local/share/libreseed/
├── lbsd.pid          # PID del daemon
├── lbsd.log          # Log del daemon
└── packages/         # Directory dei pacchetti (futura)
```

---

## Guida per Sviluppatori

### Setup Ambiente di Sviluppo

```bash
# Clona il repository
git clone https://github.com/fulgidus/libreseed.git
cd libreseed

# Verifica versione Go
go version  # Richiede Go 1.21+

# Installa dipendenze
go mod download
```

### Build per Sviluppo

```bash
# Build completa (entrambi i binari)
make build

# Build solo CLI
make build-lbs

# Build solo daemon
make build-lbsd

# Build con race detector (per testing concurrency)
make build-race
```

I binari vengono creati in `bin/`:
- `bin/lbs` — CLI per controllo daemon (8.5MB)
- `bin/lbsd` — Daemon in background (12MB)

### Testing

```bash
# Test completi
make test

# Test con coverage
make test-coverage

# Test DHT specifici
./test-dht.sh

# Test di integrazione
make test-integration

# Test con race detector
make test-race
```

### Sviluppo e Debug

```bash
# Esegui daemon in modalità verbose (foreground)
./bin/lbsd --verbose

# In un altro terminale, usa la CLI
./bin/lbs status

# Visualizza log in tempo reale
tail -f ~/.local/share/libreseed/lbsd.log

# Pulisci artifact di build
make clean

# Reinstalla dopo modifiche
make clean && make build
```

### Workflow di Sviluppo Consigliato

1. **Modifica il codice** — Edita file in `cmd/` o `pkg/`
2. **Rebuild** — `make build`
3. **Test** — `make test`
4. **Prova manualmente** — `./bin/lbs start && ./bin/lbs status`
5. **Commit** — `git add . && git commit -m "descrizione"`

### Target Makefile Utili

```bash
make help              # Mostra tutti i target disponibili
make fmt               # Formatta il codice con gofmt
make lint              # Esegue linter (golangci-lint)
make vet               # Esegue go vet per analisi statica
make checksums         # Genera SHA256SUMS
make verify            # Verifica checksum dei binari
make install-local     # Installa in bin/ locale
make install-system    # Installa in /usr/local/bin (richiede sudo)
```

### Struttura del Progetto

```
libreseed/
├── cmd/
│   ├── lbs/           # CLI source
│   │   ├── main.go
│   │   ├── start.go   # Comando 'start'
│   │   ├── stop.go    # Comando 'stop'
│   │   ├── status.go  # Comando 'status'
│   │   ├── stats.go   # Comando 'stats'
│   │   └── restart.go # Comando 'restart'
│   └── lbsd/          # Daemon source
│       └── main.go
├── pkg/
│   ├── daemon/        # Logica daemon
│   │   ├── daemon.go
│   │   ├── config.go
│   │   ├── state.go
│   │   └── statistics.go
│   ├── dht/           # Integrazione DHT BitTorrent
│   │   ├── client.go
│   │   ├── announcer.go
│   │   ├── discovery.go
│   │   └── peers.go
│   ├── crypto/        # Firma digitale pacchetti
│   │   ├── keys.go
│   │   └── signer.go
│   ├── package/       # Gestione pacchetti
│   │   ├── manifest.go
│   │   └── description.go
│   └── storage/       # Storage filesystem
│       ├── filesystem.go
│       └── metadata.go
├── Makefile           # Build automation (20+ target)
├── install.sh         # Script installazione automatica
├── test-dht.sh        # Test DHT integrazione
├── go.mod             # Dipendenze Go
└── VERSION            # Versione corrente (0.2.0)
```

### Dipendenze Principali

- **anacrolix/torrent** — Libreria BitTorrent e DHT
- **anacrolix/dht/v2** — Implementazione DHT
- **spf13/cobra** — Framework CLI (futuro)

### Debug Comune

**Problema**: `lbs start` non funziona  
**Soluzione**: Rebuild con `make clean && make build`

**Problema**: "daemon already running"  
**Soluzione**: `lbs stop` oppure rimuovi `~/.local/share/libreseed/lbsd.pid`

**Problema**: "permission denied" durante installazione  
**Soluzione**: Usa `sudo make install-system` o installa localmente con `make install-local`

**Problema**: Test DHT falliscono  
**Soluzione**: Verifica connessione internet e firewall (DHT richiede UDP)

### Contribuire

1. Fork il repository
2. Crea un branch per la feature (`git checkout -b feature/nome-feature`)
3. Commit delle modifiche (`git commit -am 'Aggiunta nuova feature'`)
4. Push al branch (`git push origin feature/nome-feature`)
5. Apri una Pull Request

### Convenzioni Codice

- **Formattazione**: Usa `make fmt` prima di ogni commit
- **Linting**: Esegui `make lint` per verificare stile
- **Testing**: Aggiungi test per nuove feature
- **Commit**: Usa [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` per nuove feature
  - `fix:` per bug fix
  - `docs:` per documentazione
  - `chore:` per task di manutenzione

---

## Architettura

LibreSeed è composto da due componenti principali:

### 1. Daemon (`lbsd`)

Il daemon gira in background e gestisce:
- **DHT Client** — Connessione alla rete DHT BitTorrent
- **Announce** — Pubblicazione dei pacchetti disponibili
- **Discovery** — Ricerca di peer per pacchetti richiesti
- **Storage** — Gestione pacchetti locali e cache

### 2. CLI (`lbs`)

L'interfaccia a riga di comando comunica con il daemon tramite:
- File PID per controllo processo
- Segnali UNIX per comandi (SIGTERM per shutdown)
- File di stato per statistiche

### Flusso di Lavoro

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│  lbs (CLI)  │────────▶│ lbsd (Daemon)│────────▶│ DHT Network │
└─────────────┘ comandi └──────────────┘ announce└─────────────┘
                                │                       │
                                │                       │
                                ▼                       ▼
                         ┌──────────────┐         ┌─────────────┐
                         │ Local Storage│         │    Peers    │
                         └──────────────┘         └─────────────┘
```

---

## Roadmap

- [x] **v0.1.0** — Struttura base progetto
- [x] **v0.2.0** — Daemon funzionante, CLI completa, integrazione DHT
- [ ] **v0.3.0** — Gestione pacchetti, manifest, firma digitale
- [ ] **v0.4.0** — Seeding e download automatico
- [ ] **v0.5.0** — API REST per integrazioni
- [ ] **v1.0.0** — Release production-ready

Vedi [CHANGELOG.md](CHANGELOG.md) per dettagli sulle release.

---

## Documentazione

- [CHANGELOG.md](CHANGELOG.md) — Storico versioni e modifiche
- [DHT_INTEGRATION_COMPLETE.md](DHT_INTEGRATION_COMPLETE.md) — Dettagli integrazione DHT
- [PROGRESS.md](PROGRESS.md) — Stato sviluppo e milestone
- [manual-test-commands.md](manual-test-commands.md) — Comandi per testing manuale

---

## Licenza

[Specificare licenza - es. MIT, GPL-3.0, Apache-2.0]

---

## Contatti

- **Repository**: https://github.com/fulgidus/libreseed
- **Issues**: https://github.com/fulgidus/libreseed/issues

---

**LibreSeed** — Distribuzione software libera e decentralizzata 🌱
