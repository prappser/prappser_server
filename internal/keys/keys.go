package keys

import (
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"

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

	// Use HKDF to create a deterministic random source
	reader := hkdf.New(sha256.New, hash[:], []byte("prappser-rsa-salt"), []byte("rsa-keypair"))

	privateKey, err := rsa.GenerateKey(&deterministicReader{reader}, KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// deterministicReader wraps an io.Reader to satisfy rand.Reader interface
type deterministicReader struct {
	reader io.Reader
}

func (d *deterministicReader) Read(p []byte) (n int, err error) {
	return d.reader.Read(p)
}
