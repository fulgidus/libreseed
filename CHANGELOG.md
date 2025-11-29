# Changelog

Tutte le modifiche rilevanti al progetto LibreSeed saranno documentate in questo file.

Il formato è basato su [Keep a Changelog](https://keepachangelog.com/it/1.0.0/),
e questo progetto aderisce al [Semantic Versioning](https://semver.org/lang/it/).

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

[0.1.0]: https://github.com/libreseed/libreseed/releases/tag/v0.1.0
