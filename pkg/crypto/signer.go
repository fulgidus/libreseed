// Package crypto fornisce operazioni crittografiche per LibreSeed.
//
// Questo package implementa le funzionalità di firma digitale utilizzando
// l'algoritmo Ed25519, inclusa la creazione, verifica e serializzazione
// delle firme digitali.
//
// Le firme digitali in LibreSeed sono utilizzate per garantire l'integrità
// e l'autenticità dei pacchetti software distribuiti attraverso la rete P2P.
package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Signature rappresenta una firma digitale Ed25519 con metadati associati.
//
// Una signature contiene la firma crittografica vera e propria insieme a
// informazioni sul firmatario, l'algoritmo utilizzato e il timestamp
// della firma.
type Signature struct {
	// Algorithm specifica l'algoritmo di firma utilizzato (sempre "Ed25519")
	Algorithm string

	// SignedBy contiene la chiave pubblica del firmatario
	SignedBy PublicKey

	// SignedData contiene la firma crittografica a 64 byte generata da Ed25519
	SignedData []byte

	// SignedAt rappresenta il momento in cui è stata creata la firma
	SignedAt time.Time
}

const (
	// Ed25519SignatureSize è la dimensione fissa di una firma Ed25519 (64 byte)
	Ed25519SignatureSize = ed25519.SignatureSize

	// AlgorithmEd25519 è l'identificatore dell'algoritmo di firma utilizzato
	AlgorithmEd25519 = "Ed25519"
)

var (
	// ErrInvalidSignatureLength viene restituito quando la lunghezza della firma non è corretta
	ErrInvalidSignatureLength = errors.New("lunghezza firma non valida: le firme Ed25519 devono essere esattamente 64 byte")

	// ErrInvalidSignature viene restituito quando la verifica della firma fallisce
	ErrInvalidSignature = errors.New("firma non valida: la verifica crittografica è fallita")

	// ErrNilPublicKey viene restituito quando viene fornita una chiave pubblica nil
	ErrNilPublicKey = errors.New("chiave pubblica nil non consentita")
)

// Sign crea una nuova firma digitale Ed25519 per i dati forniti.
//
// Parametri:
//   - privateKey: la chiave privata Ed25519 da utilizzare per la firma (64 byte)
//   - publicKey: la chiave pubblica corrispondente del firmatario
//   - data: i dati da firmare
//
// Restituisce:
//   - *Signature: la firma digitale generata con metadati
//   - error: errore in caso di parametri non validi
//
// Esempio:
//
//	privateKey := ed25519.PrivateKey(/* 64 byte */)
//	publicKey := PublicKey{/* chiave pubblica */}
//	data := []byte("dati del pacchetto da firmare")
//
//	signature, err := Sign(privateKey, publicKey, data)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Sign(privateKey ed25519.PrivateKey, publicKey PublicKey, data []byte) (*Signature, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("lunghezza chiave privata non valida: attesi %d byte, ricevuti %d byte",
			ed25519.PrivateKeySize, len(privateKey))
	}

	if publicKey.KeyBytes == nil || len(publicKey.KeyBytes) == 0 {
		return nil, ErrNilPublicKey
	}

	// Genera la firma Ed25519
	signatureBytes := ed25519.Sign(privateKey, data)

	signature := &Signature{
		Algorithm:  AlgorithmEd25519,
		SignedBy:   publicKey,
		SignedData: signatureBytes,
		SignedAt:   time.Now().UTC(),
	}

	return signature, nil
}

// Verify verifica la validità di una firma digitale utilizzando la chiave pubblica fornita.
//
// Parametri:
//   - publicKey: la chiave pubblica del firmatario da utilizzare per la verifica
//   - data: i dati originali che sono stati firmati
//   - signature: la firma da verificare
//
// Restituisce:
//   - error: nil se la firma è valida, altrimenti un errore che descrive il problema
//
// Esempio:
//
//	publicKey := PublicKey{/* chiave pubblica */}
//	data := []byte("dati del pacchetto")
//	signature := &Signature{/* firma ricevuta */}
//
//	if err := Verify(publicKey, data, signature); err != nil {
//	    log.Printf("Verifica firma fallita: %v", err)
//	    return
//	}
//	log.Println("Firma valida!")
func Verify(publicKey PublicKey, data []byte, signature *Signature) error {
	if signature == nil {
		return errors.New("signature non può essere nil")
	}

	if publicKey.KeyBytes == nil || len(publicKey.KeyBytes) == 0 {
		return ErrNilPublicKey
	}

	// Verifica che l'algoritmo sia Ed25519
	if signature.Algorithm != AlgorithmEd25519 {
		return fmt.Errorf("algoritmo non supportato: %s (solo %s è supportato)",
			signature.Algorithm, AlgorithmEd25519)
	}

	// Verifica la lunghezza della firma
	if len(signature.SignedData) != Ed25519SignatureSize {
		return ErrInvalidSignatureLength
	}

	// Esegui la verifica crittografica Ed25519
	if !ed25519.Verify(ed25519.PublicKey(publicKey.KeyBytes), data, signature.SignedData) {
		return ErrInvalidSignature
	}

	return nil
}

// SignatureFromBytes deserializza una firma da un array di byte.
//
// Parametri:
//   - data: array di byte contenente la firma (deve essere esattamente 64 byte per Ed25519)
//   - publicKey: la chiave pubblica del firmatario
//
// Restituisce:
//   - *Signature: la firma deserializzata con metadati
//   - error: errore se i dati non sono validi
//
// Nota: questa funzione assume che i byte forniti siano la firma raw Ed25519.
// Il timestamp viene impostato a zero e l'algoritmo a "Ed25519".
//
// Esempio:
//
//	signatureBytes := []byte{/* 64 byte di firma */}
//	publicKey := PublicKey{/* chiave pubblica */}
//
//	signature, err := SignatureFromBytes(signatureBytes, publicKey)
//	if err != nil {
//	    log.Fatal(err)
//	}
func SignatureFromBytes(data []byte, publicKey PublicKey) (*Signature, error) {
	if len(data) != Ed25519SignatureSize {
		return nil, ErrInvalidSignatureLength
	}

	if publicKey.KeyBytes == nil || len(publicKey.KeyBytes) == 0 {
		return nil, ErrNilPublicKey
	}

	signature := &Signature{
		Algorithm:  AlgorithmEd25519,
		SignedBy:   publicKey,
		SignedData: make([]byte, Ed25519SignatureSize),
		SignedAt:   time.Time{}, // Zero time per firme deserializzate
	}

	copy(signature.SignedData, data)

	return signature, nil
}

// Bytes restituisce i byte raw della firma (64 byte per Ed25519).
//
// Questo metodo estrae solo i dati della firma crittografica, senza
// includere i metadati come l'algoritmo, la chiave pubblica o il timestamp.
//
// Restituisce:
//   - []byte: i 64 byte della firma Ed25519
//
// Esempio:
//
//	signature := &Signature{/* firma valida */}
//	rawBytes := signature.Bytes()
//	fmt.Printf("Firma raw: %x\n", rawBytes)
func (s *Signature) Bytes() []byte {
	if s == nil || s.SignedData == nil {
		return nil
	}

	// Restituisci una copia per evitare modifiche accidentali
	result := make([]byte, len(s.SignedData))
	copy(result, s.SignedData)
	return result
}

// String restituisce una rappresentazione leggibile della firma.
//
// Il formato include l'algoritmo, una versione troncata della firma in
// esadecimale, e il timestamp (se disponibile).
//
// Restituisce:
//   - string: rappresentazione human-readable della firma
//
// Esempio di output:
//
//	Signature[Ed25519:a1b2c3d4...(64 bytes):2025-11-29T10:30:00Z]
//
// Esempio:
//
//	signature := &Signature{/* firma valida */}
//	fmt.Println(signature.String())
//	// Output: Signature[Ed25519:a1b2c3d4e5f6...(64 bytes):2025-11-29T10:30:00Z]
func (s *Signature) String() string {
	if s == nil {
		return "Signature[nil]"
	}

	var signaturePreview string
	if len(s.SignedData) > 0 {
		// Mostra i primi 8 byte in esadecimale
		preview := hex.EncodeToString(s.SignedData)
		if len(preview) > 16 {
			preview = preview[:16]
		}
		signaturePreview = fmt.Sprintf("%s...(%d bytes)", preview, len(s.SignedData))
	} else {
		signaturePreview = "empty"
	}

	var timestamp string
	if !s.SignedAt.IsZero() {
		timestamp = s.SignedAt.Format(time.RFC3339)
	} else {
		timestamp = "unknown"
	}

	return fmt.Sprintf("Signature[%s:%s:%s]",
		s.Algorithm,
		signaturePreview,
		timestamp)
}

// VerifyDualSignature verifica entrambe le firme (creator e maintainer) su un manifest.
//
// Questa funzione implementa il sistema di doppia firma richiesto per la fiducia
// dei pacchetti: sia il creatore che il maintainer devono firmare il manifest.
//
// Parametri:
//   - data: i dati originali firmati (solitamente il manifest serializzato)
//   - creatorPubKey: la chiave pubblica del creatore del pacchetto
//   - creatorSig: la firma del creatore
//   - maintainerPubKey: la chiave pubblica del maintainer del pacchetto
//   - maintainerSig: la firma del maintainer
//
// Restituisce:
//   - error: nil se entrambe le firme sono valide, altrimenti un errore descrittivo
//
// Errori possibili:
//   - Firma del creatore non valida
//   - Firma del maintainer non valida
//   - Chiavi pubbliche nil o vuote
//   - Firme nil
//
// Esempio:
//
//	manifestData := []byte("manifest serializzato")
//	creatorKey := PublicKey{/* chiave creatore */}
//	creatorSig := &Signature{/* firma creatore */}
//	maintainerKey := PublicKey{/* chiave maintainer */}
//	maintainerSig := &Signature{/* firma maintainer */}
//
//	err := VerifyDualSignature(manifestData, creatorKey, creatorSig, maintainerKey, maintainerSig)
//	if err != nil {
//	    log.Printf("Verifica doppia firma fallita: %v", err)
//	    return
//	}
func VerifyDualSignature(
	data []byte,
	creatorPubKey PublicKey,
	creatorSig *Signature,
	maintainerPubKey PublicKey,
	maintainerSig *Signature,
) error {
	// Verifica la firma del creatore
	if err := Verify(creatorPubKey, data, creatorSig); err != nil {
		return fmt.Errorf("verifica firma creatore fallita: %w", err)
	}

	// Verifica la firma del maintainer
	if err := Verify(maintainerPubKey, data, maintainerSig); err != nil {
		return fmt.Errorf("verifica firma maintainer fallita: %w", err)
	}

	return nil
}

// VerifyCreatorSignature verifica solo la firma del creatore su un manifest.
//
// Questa è una funzione helper per verificare solo la firma del creatore.
// Per il sistema completo di doppia firma, usare VerifyDualSignature.
//
// Parametri:
//   - data: i dati originali firmati
//   - creatorPubKey: la chiave pubblica del creatore
//   - signature: la firma da verificare
//
// Restituisce:
//   - error: nil se la firma è valida, altrimenti un errore
//
// Esempio:
//
//	manifestData := []byte("manifest")
//	creatorKey := PublicKey{/* chiave creatore */}
//	sig := &Signature{/* firma */}
//
//	if err := VerifyCreatorSignature(manifestData, creatorKey, sig); err != nil {
//	    log.Printf("Firma creatore non valida: %v", err)
//	}
func VerifyCreatorSignature(data []byte, creatorPubKey PublicKey, signature *Signature) error {
	if err := Verify(creatorPubKey, data, signature); err != nil {
		return fmt.Errorf("verifica firma creatore fallita: %w", err)
	}
	return nil
}

// VerifyMaintainerSignature verifica solo la firma del maintainer su un manifest.
//
// Questa è una funzione helper per verificare solo la firma del maintainer.
// Per il sistema completo di doppia firma, usare VerifyDualSignature.
//
// Parametri:
//   - data: i dati originali firmati
//   - maintainerPubKey: la chiave pubblica del maintainer
//   - signature: la firma da verificare
//
// Restituisce:
//   - error: nil se la firma è valida, altrimenti un errore
//
// Esempio:
//
//	manifestData := []byte("manifest")
//	maintainerKey := PublicKey{/* chiave maintainer */}
//	sig := &Signature{/* firma */}
//
//	if err := VerifyMaintainerSignature(manifestData, maintainerKey, sig); err != nil {
//	    log.Printf("Firma maintainer non valida: %v", err)
//	}
func VerifyMaintainerSignature(data []byte, maintainerPubKey PublicKey, signature *Signature) error {
	if err := Verify(maintainerPubKey, data, signature); err != nil {
		return fmt.Errorf("verifica firma maintainer fallita: %w", err)
	}
	return nil
}
