// Package crypto fornisce le operazioni crittografiche fondamentali per LibreSeed,
// inclusa la gestione delle chiavi pubbliche Ed25519 e le operazioni di verifica delle firme.
//
// LibreSeed utilizza Ed25519 per tutte le operazioni di firma e verifica, garantendo
// sicurezza crittografica e prestazioni elevate. Le chiavi pubbliche vengono identificate
// tramite fingerprint SHA-256 per facilitare la gestione e il riferimento leggibile.
package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// PublicKey rappresenta una chiave pubblica Ed25519 utilizzata per la verifica
// delle firme nei pacchetti LibreSeed.
//
// Il sistema dual-signature di LibreSeed richiede che tutte le firme (sia del
// manifest completo che del minimal descriptor) siano verificabili tramite la
// stessa chiave pubblica del creatore, garantendo autenticità e integrità.
//
// Example:
//
//	// Crea una nuova chiave pubblica da bytes raw
//	pubKey, err := crypto.NewPublicKey(keyBytes)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Ottieni il fingerprint per identificazione
//	fp := pubKey.Fingerprint()
//	fmt.Printf("Fingerprint: %s\n", fp)
//
//	// Verifica una firma
//	valid := pubKey.Verify(message, signature)
//	if !valid {
//		log.Fatal("signature verification failed")
//	}
type PublicKey struct {
	// Algorithm identifica l'algoritmo crittografico utilizzato.
	// Per LibreSeed v1.x, questo valore è sempre "ed25519".
	Algorithm string

	// KeyBytes contiene la rappresentazione raw della chiave pubblica Ed25519.
	// Per Ed25519, questo array deve essere esattamente 32 bytes.
	KeyBytes []byte
}

// NewPublicKey crea una nuova istanza di PublicKey dai bytes raw forniti.
//
// La funzione valida che i bytes forniti costituiscano una chiave Ed25519 valida
// di lunghezza corretta (32 bytes). Se la validazione fallisce, viene restituito
// un errore descrittivo.
//
// Parameters:
//   - keyBytes: Raw bytes della chiave pubblica Ed25519 (deve essere 32 bytes)
//
// Returns:
//   - *PublicKey: Istanza inizializzata e validata
//   - error: Errore se keyBytes è nil, vuoto o non ha lunghezza corretta
//
// Example:
//
//	keyBytes := []byte{...} // 32 bytes
//	pubKey, err := crypto.NewPublicKey(keyBytes)
//	if err != nil {
//		return fmt.Errorf("invalid public key: %w", err)
//	}
func NewPublicKey(keyBytes []byte) (*PublicKey, error) {
	if keyBytes == nil {
		return nil, fmt.Errorf("key bytes cannot be nil")
	}

	if len(keyBytes) == 0 {
		return nil, fmt.Errorf("key bytes cannot be empty")
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: expected %d bytes, got %d",
			ed25519.PublicKeySize, len(keyBytes))
	}

	return &PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  keyBytes,
	}, nil
}

// Fingerprint calcola e restituisce il fingerprint della chiave pubblica.
//
// Il fingerprint è definito come i primi 8 bytes dell'hash SHA-256 della chiave
// pubblica, codificati in esadecimale. Questo produce una stringa di 16 caratteri
// leggibile e facile da comunicare per identificare univocamente le chiavi.
//
// Il fingerprint viene utilizzato in LibreSeed per:
//   - Identificazione rapida delle chiavi nei log e nell'interfaccia utente
//   - Verifica manuale dell'identità del creatore
//   - Riferimenti compatti nelle strutture dati
//
// Returns:
//   - string: Fingerprint di 16 caratteri esadecimali (8 bytes)
//
// Example:
//
//	fp := pubKey.Fingerprint()
//	fmt.Printf("Key fingerprint: %s\n", fp) // Output: "a1b2c3d4e5f67890"
func (pk *PublicKey) Fingerprint() string {
	hash := sha256.Sum256(pk.KeyBytes)
	// Prendi i primi 8 bytes dell'hash (64 bits)
	fingerprint := hash[:8]
	// Codifica in esadecimale (16 caratteri)
	return hex.EncodeToString(fingerprint)
}

// Verify verifica una firma Ed25519 utilizzando questa chiave pubblica.
//
// Implementa la verifica standard Ed25519 come definita in RFC 8032.
// La funzione è utilizzata in LibreSeed per verificare sia le firme del
// manifest completo che quelle del minimal descriptor, garantendo che
// entrambe provengano dallo stesso creatore.
//
// La verifica fallisce se:
//   - La firma non corrisponde al messaggio
//   - La firma è stata alterata
//   - La firma non è stata creata dalla chiave privata corrispondente
//
// Parameters:
//   - message: Il messaggio originale che è stato firmato
//   - signature: La firma Ed25519 da verificare (deve essere 64 bytes)
//
// Returns:
//   - bool: true se la firma è valida, false altrimenti
//
// Example:
//
//	message := []byte("hello world")
//	signature := []byte{...} // 64 bytes firma Ed25519
//
//	if pubKey.Verify(message, signature) {
//		fmt.Println("Signature is valid")
//	} else {
//		fmt.Println("Signature verification failed")
//	}
//
// Note:
//   - Questa funzione non restituisce errori; una verifica fallita restituisce false
//   - Per Ed25519, la firma deve essere esattamente 64 bytes
//   - La funzione è sicura contro timing attacks
func (pk *PublicKey) Verify(message []byte, signature []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(pk.KeyBytes), message, signature)
}

// Bytes restituisce la rappresentazione raw della chiave pubblica.
//
// Questa funzione fornisce accesso diretto ai bytes della chiave per
// operazioni di serializzazione, storage o trasmissione in rete.
//
// Returns:
//   - []byte: Raw bytes della chiave pubblica Ed25519 (32 bytes)
//
// Example:
//
//	bytes := pubKey.Bytes()
//	// Salva bytes su file o invia in rete
//	err := os.WriteFile("pubkey.bin", bytes, 0600)
func (pk *PublicKey) Bytes() []byte {
	return pk.KeyBytes
}

// String fornisce una rappresentazione leggibile della chiave pubblica.
//
// Il formato restituito è "algorithm:fingerprint", dove:
//   - algorithm è sempre "ed25519" per LibreSeed v1.x
//   - fingerprint è la stringa esadecimale di 16 caratteri
//
// Questo formato è utile per log, debug e visualizzazione nell'interfaccia
// utente, fornendo un modo compatto ma informativo per identificare le chiavi.
//
// Returns:
//   - string: Rappresentazione nel formato "algorithm:fingerprint"
//
// Example:
//
//	fmt.Println(pubKey.String())
//	// Output: "ed25519:a1b2c3d4e5f67890"
func (pk *PublicKey) String() string {
	return fmt.Sprintf("%s:%s", pk.Algorithm, pk.Fingerprint())
}
