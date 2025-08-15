package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

const (
	PrivateKeyPath = "files/server_private_key.pem"
	PublicKeyPath  = "files/server_public_key.pem"
	KeySize        = 2048
)

// GenerateRSAKeyPair generates a new RSA key pair and saves it to files
func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(PrivateKeyPath), 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Save private key
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	if err := os.WriteFile(PrivateKeyPath, pem.EncodeToMemory(privateKeyPEM), 0600); err != nil {
		return nil, nil, fmt.Errorf("failed to save private key: %w", err)
	}

	// Save public key
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	}
	if err := os.WriteFile(PublicKeyPath, pem.EncodeToMemory(publicKeyPEM), 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save public key: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// LoadRSAKeyPair loads existing RSA key pair from files
func LoadRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Check if private key exists
	if _, err := os.Stat(PrivateKeyPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("private key file not found")
	}

	// Load private key
	privateKeyBytes, err := os.ReadFile(PrivateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// GetOrGenerateRSAKeyPair loads existing keys or generates new ones
func GetOrGenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Try to load existing keys
	privateKey, publicKey, err := LoadRSAKeyPair()
	if err == nil {
		return privateKey, publicKey, nil
	}

	// If keys don't exist, generate new ones
	fmt.Println("RSA key pair not found. Generating new keys...")
	return GenerateRSAKeyPair()
}
