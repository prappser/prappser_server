package keys

import (
	"crypto/rsa"
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/hkdf"
)

const KeySize = 2048

// DeriveRSAKeyPair deterministically generates RSA keys from a seed.
// Same seed always produces the same key pair.
func DeriveRSAKeyPair(masterPassword, externalURL string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if masterPassword == "" {
		return nil, nil, fmt.Errorf("master password is required for key derivation")
	}
	if externalURL == "" {
		return nil, nil, fmt.Errorf("external URL is required for key derivation")
	}

	// Combine password and URL to create unique seed per deployment
	seed := masterPassword + externalURL
	hash := sha256.Sum256([]byte(seed))

	// Use HKDF to derive a 32-byte key and 12-byte nonce for ChaCha20
	hkdfReader := hkdf.New(sha256.New, hash[:], []byte("prappser-rsa-salt"), []byte("rsa-keypair"))

	key := make([]byte, 32)
	nonce := make([]byte, 12)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, nil, fmt.Errorf("failed to derive key: %w", err)
	}
	if _, err := hkdfReader.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to derive nonce: %w", err)
	}

	// Use ChaCha20 as a CSPRNG - it can produce unlimited random bytes
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ChaCha20 cipher: %w", err)
	}

	// Create a deterministic reader using ChaCha20 keystream
	reader := &chachaReader{cipher: cipher}

	privateKey, err := rsa.GenerateKey(reader, KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// chachaReader generates deterministic random bytes using ChaCha20 keystream
type chachaReader struct {
	cipher *chacha20.Cipher
}

func (c *chachaReader) Read(p []byte) (n int, err error) {
	// XOR zeros with keystream to get pure keystream output
	for i := range p {
		p[i] = 0
	}
	c.cipher.XORKeyStream(p, p)
	return len(p), nil
}
